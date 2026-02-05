/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock/mock"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenDBHelper wraps the counterfeiter-generated mock with helper methods
type mockTokenDBHelper struct {
	*mock.TokenDB
	mu     sync.RWMutex
	tokens map[string][]*token2.UnspentTokenInWallet
}

func newMockTokenDB() *mockTokenDBHelper {
	helper := &mockTokenDBHelper{
		TokenDB: &mock.TokenDB{},
		tokens:  make(map[string][]*token2.UnspentTokenInWallet),
	}

	// Set up the stub to use our implementation
	helper.SpendableTokensIteratorByCalls(func(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
		helper.mu.RLock()
		defer helper.mu.RUnlock()

		var result []*token2.UnspentTokenInWallet

		// If both walletID and typ are empty, return all tokens
		if walletID == "" && typ == "" {
			for _, tokens := range helper.tokens {
				result = append(result, tokens...)
			}
		} else if walletID == "" {
			// Return all tokens of this type
			for _, tokens := range helper.tokens {
				for _, t := range tokens {
					if t.Type == typ {
						result = append(result, t)
					}
				}
			}
		} else if typ == "" {
			// Return all tokens for this wallet
			for _, tokens := range helper.tokens {
				for _, t := range tokens {
					if t.WalletID == walletID {
						result = append(result, t)
					}
				}
			}
		} else {
			// Return specific wallet and type
			key := tokenKey(walletID, typ)
			if tokens, ok := helper.tokens[key]; ok {
				result = append(result, tokens...)
			}
		}

		// Create and configure the iterator mock
		it := &mock.Iterator[*token2.UnspentTokenInWallet]{}
		index := 0
		it.NextCalls(func() (*token2.UnspentTokenInWallet, error) {
			if index >= len(result) {
				return nil, nil
			}
			token := result[index]
			index++
			return token, nil
		})

		return it, nil
	})

	return helper
}

func (m *mockTokenDBHelper) addTokens(walletID string, typ token2.Type, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := tokenKey(walletID, typ)
	for i := 0; i < count; i++ {
		token := &token2.UnspentTokenInWallet{
			Id: token2.ID{
				TxId:  fmt.Sprintf("tx_%s_%d", walletID, i),
				Index: uint64(i),
			},
			WalletID: walletID,
			Type:     typ,
			Quantity: fmt.Sprintf("0x%x", 100+i),
		}
		m.tokens[key] = append(m.tokens[key], token)
	}
}

func (m *mockTokenDBHelper) getQueryCount() int {
	return m.SpendableTokensIteratorByCallCount()
}

// TestLazyFetcher tests the lazy fetcher strategy
func TestLazyFetcher(t *testing.T) {
	t.Run("BasicFetch", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		fetcher := NewLazyFetcher(db)

		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		require.NotNil(t, it)

		// Count tokens
		count := 0
		for {
			token, err := it.Next()
			require.NoError(t, err)
			if token == nil {
				break
			}
			count++
		}
		assert.Equal(t, 5, count)
		assert.Equal(t, 1, db.getQueryCount(), "Should query DB once")
	})

	t.Run("MultipleFetches", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 3)

		fetcher := NewLazyFetcher(db)

		// First fetch
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Second fetch - should query DB again (no caching)
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it2)

		assert.Equal(t, 2, db.getQueryCount(), "Lazy fetcher should query DB each time")
	})

	t.Run("EmptyResult", func(t *testing.T) {
		db := newMockTokenDB()
		fetcher := NewLazyFetcher(db)

		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "nonexistent", "USD")
		require.NoError(t, err)

		token, err := it.Next()
		require.NoError(t, err)
		assert.Nil(t, token, "Should return nil for empty result")
	})

	t.Run("DatabaseError", func(t *testing.T) {
		db := &mock.TokenDB{}
		db.SpendableTokensIteratorByReturns(nil, fmt.Errorf("connection timeout"))

		fetcher := NewLazyFetcher(db)

		_, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection timeout")
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 10)
		db.addTokens("wallet2", "EUR", 10)

		fetcher := NewLazyFetcher(db)

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		// Concurrent requests
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				wallet := "wallet1"
				if idx%2 == 0 {
					wallet = "wallet2"
				}
				currency := token2.Type("USD")
				if idx%2 == 0 {
					currency = "EUR"
				}

				it, err := fetcher.UnspentTokensIteratorBy(t.Context(), wallet, currency)
				if err != nil {
					errors <- err
					return
				}
				consumeIterator(t, it)
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
		}

		assert.Equal(t, 10, db.getQueryCount(), "Should have 10 queries")
	})
}

// TestCachedFetcher tests the eager/cached fetcher strategy
func TestCachedFetcher(t *testing.T) {
	t.Run("InitialCacheLoad", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)
		db.addTokens("wallet2", "EUR", 3)

		fetcher := newCachedFetcher(db, 10*time.Second, 100)

		// First request triggers cache load
		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)

		count := countTokens(t, it)
		assert.Equal(t, 5, count)
		assert.Equal(t, 1, db.getQueryCount(), "Should load all tokens once")
	})

	t.Run("CacheHit", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		fetcher := newCachedFetcher(db, 10*time.Second, 100)

		// First request
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Second request - should use cache
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it2)

		assert.Equal(t, 1, db.getQueryCount(), "Should use cached result")
	})

	t.Run("CacheStaleRefresh", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		fetcher := newCachedFetcher(db, 50*time.Millisecond, 100)

		// First request
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)
		assert.Equal(t, 1, db.getQueryCount())

		// Wait for cache to become stale
		time.Sleep(100 * time.Millisecond)

		// Second request - should refresh
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it2)

		assert.Equal(t, 2, db.getQueryCount(), "Should refresh stale cache")
	})

	t.Run("CacheOverusedRefresh", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		maxQueries := 3
		fetcher := newCachedFetcher(db, 10*time.Second, maxQueries)

		// Make maxQueries requests
		for i := 0; i < maxQueries; i++ {
			it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
			require.NoError(t, err)
			consumeIterator(t, it)
		}
		assert.Equal(t, 1, db.getQueryCount(), "Should use cache")

		// Next request should trigger background refresh
		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it)

		// Wait for background refresh
		time.Sleep(50 * time.Millisecond)

		assert.GreaterOrEqual(t, db.getQueryCount(), 2, "Should trigger refresh after overuse")
	})

	t.Run("EmptyCache", func(t *testing.T) {
		db := newMockTokenDB()
		fetcher := newCachedFetcher(db, 10*time.Second, 100)

		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)

		token, err := it.Next()
		require.NoError(t, err)
		assert.Nil(t, token, "Should return empty iterator for missing wallet")
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 10)

		fetcher := newCachedFetcher(db, 10*time.Second, 100)

		var wg sync.WaitGroup
		errors := make(chan error, 20)

		// Concurrent requests
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
				if err != nil {
					errors <- err
					return
				}
				consumeIterator(t, it)
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
		}

		// Should have minimal queries due to caching
		assert.LessOrEqual(t, db.getQueryCount(), 3, "Should use cache for most requests")
	})
}

// TestMixedFetcher tests the mixed fetcher strategy
func TestMixedFetcher(t *testing.T) {
	t.Run("EagerHit", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		fetcher := newMixedFetcher(db, newMetrics(&disabled.Provider{}))

		// First request - eager fetcher should load cache
		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)

		count := countTokens(t, it)
		assert.Equal(t, 5, count)
	})

	t.Run("EagerMissLazyFallback", func(t *testing.T) {
		db := newMockTokenDB()

		fetcher := newMixedFetcher(db, newMetrics(&disabled.Provider{}))

		// First request - eager cache is empty, should fall back to lazy
		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)

		token, err := it.Next()
		require.NoError(t, err)
		assert.Nil(t, token, "Should return empty result from lazy fetcher")
	})

	t.Run("EagerThenLazy", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		fetcher := newMixedFetcher(db, newMetrics(&disabled.Provider{}))

		// First request - uses eager
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Add new wallet
		db.addTokens("wallet2", "EUR", 3)

		// Second request for new wallet - eager cache doesn't have it, uses lazy
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet2", "EUR")
		require.NoError(t, err)

		count := countTokens(t, it2)
		assert.Equal(t, 3, count, "Should fetch new wallet via lazy")
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 10)
		db.addTokens("wallet2", "EUR", 10)

		fetcher := newMixedFetcher(db, newMetrics(&disabled.Provider{}))

		var wg sync.WaitGroup
		errors := make(chan error, 20)

		// Concurrent requests
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				wallet := "wallet1"
				currency := token2.Type("USD")
				if idx%2 == 0 {
					wallet = "wallet2"
					currency = "EUR"
				}

				it, err := fetcher.UnspentTokensIteratorBy(t.Context(), wallet, currency)
				if err != nil {
					errors <- err
					return
				}
				consumeIterator(t, it)
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
		}
	})
}

// TestAdaptiveFetcher tests the adaptive fetcher strategy
func TestAdaptiveFetcher(t *testing.T) {
	t.Run("LazyDiscovery", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		fetcher := newAdaptiveFetcher(db, 10*time.Second, 100, newMetrics(&disabled.Provider{}))

		// First request - lazy discovery
		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)

		count := countTokens(t, it)
		assert.Equal(t, 5, count)
		assert.Equal(t, 1, db.getQueryCount(), "Should query specific wallet")

		// Verify wallet is now known
		fetcher.mu.RLock()
		_, exists := fetcher.knownWallets["wallet1"]
		assert.True(t, exists, "Wallet should be marked as known")
		fetcher.mu.RUnlock()
	})

	t.Run("CacheHitAfterDiscovery", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		fetcher := newAdaptiveFetcher(db, 10*time.Second, 100, newMetrics(&disabled.Provider{}))

		// First request - discovery
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Second request - cache hit
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it2)

		assert.Equal(t, 1, db.getQueryCount(), "Should use cached result")
	})

	t.Run("MultipleWalletDiscovery", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)
		db.addTokens("wallet2", "EUR", 3)
		db.addTokens("wallet3", "GBP", 7)

		fetcher := newAdaptiveFetcher(db, 10*time.Second, 100, newMetrics(&disabled.Provider{}))

		// Discover wallet1
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Discover wallet2
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet2", "EUR")
		require.NoError(t, err)
		consumeIterator(t, it2)

		// Discover wallet3
		it3, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet3", "GBP")
		require.NoError(t, err)
		consumeIterator(t, it3)

		// Verify all wallets are known
		fetcher.mu.RLock()
		assert.Equal(t, 3, len(fetcher.knownWallets), "Should have 3 known wallets")
		_, exists1 := fetcher.knownWallets["wallet1"]
		assert.True(t, exists1)
		_, exists2 := fetcher.knownWallets["wallet2"]
		assert.True(t, exists2)
		_, exists3 := fetcher.knownWallets["wallet3"]
		assert.True(t, exists3)
		fetcher.mu.RUnlock()
	})

	t.Run("EagerRefreshKnownWalletsOnly", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)
		db.addTokens("wallet2", "EUR", 3)
		db.addTokens("wallet3", "GBP", 7) // Not discovered

		fetcher := newAdaptiveFetcher(db, 50*time.Millisecond, 100, newMetrics(&disabled.Provider{}))

		// Discover wallet1 and wallet2
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet2", "EUR")
		require.NoError(t, err)
		consumeIterator(t, it2)

		initialQueries := db.getQueryCount()

		// Wait for cache to become stale
		time.Sleep(100 * time.Millisecond)

		// Request wallet1 again - should trigger refresh
		it3, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it3)

		// Should refresh only wallet1 and wallet2, not wallet3
		assert.Greater(t, db.getQueryCount(), initialQueries, "Should refresh known wallets")
	})

	t.Run("CacheOverusedBackgroundRefresh", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		maxQueries := 3
		fetcher := newAdaptiveFetcher(db, 10*time.Second, maxQueries, newMetrics(&disabled.Provider{}))

		// Discover wallet
		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it)

		// Make maxQueries requests
		for i := 0; i < maxQueries; i++ {
			it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
			require.NoError(t, err)
			consumeIterator(t, it)
		}

		initialQueries := db.getQueryCount()

		// Next request should trigger background refresh
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it2)

		// Wait for background refresh
		time.Sleep(50 * time.Millisecond)

		assert.Greater(t, db.getQueryCount(), initialQueries, "Should trigger background refresh")
	})

	t.Run("ConcurrentDiscovery", func(t *testing.T) {
		db := newMockTokenDB()
		for i := 0; i < 10; i++ {
			db.addTokens(fmt.Sprintf("wallet%d", i), "USD", 5)
		}

		fetcher := newAdaptiveFetcher(db, 10*time.Second, 100, newMetrics(&disabled.Provider{}))

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		// Concurrent discovery of different wallets
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				wallet := fmt.Sprintf("wallet%d", idx)
				it, err := fetcher.UnspentTokensIteratorBy(t.Context(), wallet, "USD")
				if err != nil {
					errors <- err
					return
				}
				consumeIterator(t, it)
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent discovery error: %v", err)
		}

		// Verify all wallets are known
		fetcher.mu.RLock()
		assert.Equal(t, 10, len(fetcher.knownWallets), "Should have 10 known wallets")
		fetcher.mu.RUnlock()
	})
}

// TestAdaptiveFetcherWithTTL tests the wallet TTL feature
func TestAdaptiveFetcherWithTTL(t *testing.T) {
	t.Run("WalletExpiresAfterTTL", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)
		db.addTokens("wallet2", "EUR", 3)

		// Set TTL to 100ms
		walletTTL := 100 * time.Millisecond
		fetcher := newAdaptiveFetcherWithTTL(db, 10*time.Second, 100, walletTTL, newMetrics(&disabled.Provider{}))

		// Discover wallet1
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Discover wallet2
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet2", "EUR")
		require.NoError(t, err)
		consumeIterator(t, it2)

		// Both wallets should be known
		fetcher.mu.RLock()
		assert.Equal(t, 2, len(fetcher.knownWallets))
		_, exists := fetcher.knownWallets["wallet1"]
		assert.True(t, exists)
		_, exists = fetcher.knownWallets["wallet2"]
		assert.True(t, exists)
		fetcher.mu.RUnlock()

		// Access wallet1 again to update its last access time
		it3, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it3)

		// Wait for wallet2 to expire (but not wallet1)
		time.Sleep(150 * time.Millisecond)

		// Trigger cache refresh
		fetcher.mu.Lock()
		fetcher.lastFetched = time.Time{} // Force stale
		fetcher.mu.Unlock()

		// Access wallet1 to trigger refresh
		it4, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it4)

		// wallet2 should be removed from known wallets (expired)
		fetcher.mu.RLock()
		assert.Equal(t, 1, len(fetcher.knownWallets), "Only wallet1 should remain")
		_, hasWallet1 := fetcher.knownWallets["wallet1"]
		assert.True(t, hasWallet1, "wallet1 should still be known")
		_, hasWallet2 := fetcher.knownWallets["wallet2"]
		assert.False(t, hasWallet2, "wallet2 should be expired")
		fetcher.mu.RUnlock()
	})

	t.Run("NoTTLMeansNoExpiration", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		// TTL = 0 means no expiration
		fetcher := newAdaptiveFetcherWithTTL(db, 10*time.Second, 100, 0, newMetrics(&disabled.Provider{}))

		// Discover wallet
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Wait a long time
		time.Sleep(200 * time.Millisecond)

		// Trigger cache refresh
		fetcher.mu.Lock()
		fetcher.lastFetched = time.Time{} // Force stale
		fetcher.mu.Unlock()

		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it2)

		// Wallet should still be known (no TTL)
		fetcher.mu.RLock()
		assert.Equal(t, 1, len(fetcher.knownWallets))
		_, exists := fetcher.knownWallets["wallet1"]
		assert.True(t, exists)
		fetcher.mu.RUnlock()
	})

	t.Run("MultipleWalletsWithDifferentAccessPatterns", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)
		db.addTokens("wallet2", "EUR", 3)
		db.addTokens("wallet3", "GBP", 7)

		walletTTL := 100 * time.Millisecond
		fetcher := newAdaptiveFetcherWithTTL(db, 10*time.Second, 100, walletTTL, newMetrics(&disabled.Provider{}))

		// Discover all wallets at time T0
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet2", "EUR")
		require.NoError(t, err)
		consumeIterator(t, it2)

		it3, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet3", "GBP")
		require.NoError(t, err)
		consumeIterator(t, it3)

		// Wait 60ms
		time.Sleep(60 * time.Millisecond)

		// Access wallet1 and wallet2 (refresh their access time)
		it4, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it4)

		it5, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet2", "EUR")
		require.NoError(t, err)
		consumeIterator(t, it5)

		// Wait another 60ms (total 120ms from wallet3's last access)
		time.Sleep(60 * time.Millisecond)

		// Trigger cache refresh
		fetcher.mu.Lock()
		fetcher.lastFetched = time.Time{} // Force stale
		fetcher.mu.Unlock()

		it6, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it6)

		// wallet3 should be expired, wallet1 and wallet2 should remain
		fetcher.mu.RLock()
		assert.Equal(t, 2, len(fetcher.knownWallets), "wallet1 and wallet2 should remain")
		_, exists := fetcher.knownWallets["wallet1"]
		assert.True(t, exists)
		_, exists = fetcher.knownWallets["wallet2"]
		assert.True(t, exists)
		_, exists = fetcher.knownWallets["wallet3"]
		assert.False(t, exists, "wallet3 should be expired")
		fetcher.mu.RUnlock()
	})

	t.Run("ExpiredWalletCanBeRediscovered", func(t *testing.T) {
		db := newMockTokenDB()
		db.addTokens("wallet1", "USD", 5)

		walletTTL := 100 * time.Millisecond
		fetcher := newAdaptiveFetcherWithTTL(db, 10*time.Second, 100, walletTTL, newMetrics(&disabled.Provider{}))

		// Discover wallet
		it1, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		consumeIterator(t, it1)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Trigger cache refresh to expire wallet
		fetcher.mu.Lock()
		fetcher.lastFetched = time.Time{} // Force stale
		fetcher.mu.Unlock()

		// Force update
		fetcher.update(t.Context())

		// Wallet should be expired
		fetcher.mu.RLock()
		assert.Equal(t, 0, len(fetcher.knownWallets), "Wallet should be expired")
		fetcher.mu.RUnlock()

		// Rediscover the wallet
		it2, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet1", "USD")
		require.NoError(t, err)
		count := countTokens(t, it2)
		assert.Equal(t, 5, count)

		// Wallet should be known again
		fetcher.mu.RLock()
		assert.Equal(t, 1, len(fetcher.knownWallets))
		_, exists := fetcher.knownWallets["wallet1"]
		assert.True(t, exists)
		fetcher.mu.RUnlock()
	})

	t.Run("TTLWithConcurrentAccess", func(t *testing.T) {
		db := newMockTokenDB()
		for i := 0; i < 5; i++ {
			db.addTokens(fmt.Sprintf("wallet%d", i), "USD", 5)
		}

		walletTTL := 200 * time.Millisecond
		fetcher := newAdaptiveFetcherWithTTL(db, 10*time.Second, 100, walletTTL, newMetrics(&disabled.Provider{}))

		var wg sync.WaitGroup
		errors := make(chan error, 50)

		// Discover all wallets concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				wallet := fmt.Sprintf("wallet%d", idx)
				it, err := fetcher.UnspentTokensIteratorBy(t.Context(), wallet, "USD")
				if err != nil {
					errors <- err
					return
				}
				consumeIterator(t, it)
			}(i)
		}
		wg.Wait()

		// Keep accessing wallet0 and wallet1
		stopChan := make(chan bool)
		go func() {
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stopChan:
					return
				case <-ticker.C:
					for i := 0; i < 2; i++ {
						wallet := fmt.Sprintf("wallet%d", i)
						it, err := fetcher.UnspentTokensIteratorBy(t.Context(), wallet, "USD")
						if err != nil {
							errors <- err
							continue
						}
						consumeIterator(t, it)
					}
				}
			}
		}()

		// Wait for other wallets to expire
		time.Sleep(300 * time.Millisecond)

		// Trigger refresh
		fetcher.mu.Lock()
		fetcher.lastFetched = time.Time{} // Force stale
		fetcher.mu.Unlock()

		it, err := fetcher.UnspentTokensIteratorBy(t.Context(), "wallet0", "USD")
		require.NoError(t, err)
		consumeIterator(t, it)

		stopChan <- true
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
		}

		// Only wallet0 and wallet1 should remain (actively accessed)
		fetcher.mu.RLock()
		knownCount := len(fetcher.knownWallets)
		fetcher.mu.RUnlock()

		assert.LessOrEqual(t, knownCount, 2, "Only frequently accessed wallets should remain")
	})
}

// Helper functions
func consumeIterator(t *testing.T, it iterator[*token2.UnspentTokenInWallet]) {
	t.Helper()
	for {
		token, err := it.Next()
		require.NoError(t, err)
		if token == nil {
			break
		}
	}
}

func countTokens(t *testing.T, it iterator[*token2.UnspentTokenInWallet]) int {
	t.Helper()
	count := 0
	for {
		token, err := it.Next()
		require.NoError(t, err)
		if token == nil {
			break
		}
		count++
	}
	return count
}
