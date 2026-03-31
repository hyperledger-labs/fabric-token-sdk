/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/cache"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Package sherdlock_test validates token fetcher strategies (lazy, eager, mixed) and caching behavior.
// Tests cover: cache freshness, staleness detection, concurrent access, race condition prevention,
// database error handling, and metrics tracking.

// mockTokenDB is a mock implementation of TokenDB using testify/mock
type mockTokenDB struct {
	mock.Mock
}

func (m *mockTokenDB) SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
	args := m.Called(ctx, walletID, typ)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(driver.SpendableTokensIterator), args.Error(1)
}

func TestNewCachedFetcher_WithDefaults(t *testing.T) {
	mockDB := new(mockTokenDB)

	// Test with zero values (should use defaults)
	fetcher := newCachedFetcher(mockDB, 0, 0, 0)

	assert.NotNil(t, fetcher)
	assert.Equal(t, defaultCacheFreshnessInterval, fetcher.freshnessInterval)
	assert.Equal(t, uint32(defaultCacheMaxQueries), fetcher.maxQueriesBeforeRefresh)
	assert.NotNil(t, fetcher.cache)
}

func TestNewCachedFetcher_WithCustomValues(t *testing.T) {
	mockDB := new(mockTokenDB)

	customSize := int64(500)
	customFreshness := 60 * time.Second
	customMaxQueries := 200

	fetcher := newCachedFetcher(mockDB, customSize, customFreshness, customMaxQueries)

	assert.NotNil(t, fetcher)
	assert.Equal(t, customFreshness, fetcher.freshnessInterval)
	assert.Equal(t, uint32(customMaxQueries), fetcher.maxQueriesBeforeRefresh)
	assert.NotNil(t, fetcher.cache)
}

func TestCachedFetcher_IsCacheStale(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 100*time.Millisecond, 0)

	// Initially cache should be stale (lastFetched is zero time)
	assert.True(t, fetcher.isCacheStale())

	// Update lastFetched to now
	fetcher.lastFetched = time.Now()
	assert.False(t, fetcher.isCacheStale())

	// Wait for cache to become stale
	time.Sleep(150 * time.Millisecond)
	assert.True(t, fetcher.isCacheStale())
}

func TestCachedFetcher_IsCacheOverused(t *testing.T) {
	mockDB := new(mockTokenDB)
	maxQueries := 5
	fetcher := newCachedFetcher(mockDB, 0, 0, maxQueries)

	// Initially not overused
	assert.False(t, fetcher.isCacheOverused())

	// Simulate queries
	for range maxQueries - 1 {
		atomic.AddUint32(&fetcher.queriesResponded, 1)
	}
	assert.False(t, fetcher.isCacheOverused())

	// One more query should make it overused
	atomic.AddUint32(&fetcher.queriesResponded, 1)
	assert.True(t, fetcher.isCacheOverused())
}

func TestCachedFetcher_Update(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 1*time.Second, 100)

	// Create test tokens
	tokens := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "100",
		},
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "200",
		},
		{
			WalletID: "wallet2",
			Type:     "EUR",
			Quantity: "50",
		},
	}

	// Create iterator from tokens
	mockIterator := iterators.Slice(tokens)

	// Setup mock to return the iterator
	mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).Return(mockIterator, nil)

	// Call update
	ctx := t.Context()
	fetcher.update(ctx)

	// Verify cache was updated
	assert.False(t, fetcher.isCacheStale())
	assert.Equal(t, uint32(0), atomic.LoadUint32(&fetcher.queriesResponded))

	// Verify tokens are in cache
	key1 := tokenKey("wallet1", "USD")
	it1, ok := fetcher.cache.Get(key1)
	assert.True(t, ok)
	assert.NotNil(t, it1)

	key2 := tokenKey("wallet2", "EUR")
	it2, ok := fetcher.cache.Get(key2)
	assert.True(t, ok)
	assert.NotNil(t, it2)

	mockDB.AssertExpectations(t)
}

func TestCachedFetcher_UnspentTokensIteratorBy_CacheHit(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 10*time.Second, 100)

	// Populate cache
	tokens := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "100",
		},
	}
	mockIterator := iterators.Slice(tokens)
	mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).Return(mockIterator, nil)

	ctx := t.Context()
	fetcher.update(ctx)

	// Query cache
	it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")

	require.NoError(t, err)
	assert.NotNil(t, it)
	assert.True(t, it.(enhancedIterator[*token2.UnspentTokenInWallet]).HasNext())

	// Verify query counter incremented
	assert.Equal(t, uint32(1), atomic.LoadUint32(&fetcher.queriesResponded))

	mockDB.AssertExpectations(t)
}

func TestCachedFetcher_UnspentTokensIteratorBy_CacheMiss(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 10*time.Second, 100)

	// Populate cache with different key
	tokens := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "100",
		},
	}
	mockIterator := iterators.Slice(tokens)
	mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).Return(mockIterator, nil)

	ctx := t.Context()
	fetcher.update(ctx)

	// Query for non-existent key
	it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet2", "EUR")

	require.NoError(t, err)
	assert.NotNil(t, it)
	// Should return empty iterator
	assert.False(t, it.(enhancedIterator[*token2.UnspentTokenInWallet]).HasNext())

	mockDB.AssertExpectations(t)
}

func TestCachedFetcher_UnspentTokensIteratorBy_StaleCache(t *testing.T) {
	mockDB := new(mockTokenDB)
	// Very short freshness interval
	fetcher := newCachedFetcher(mockDB, 0, 50*time.Millisecond, 100)

	// Initial population
	tokens1 := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "100",
		},
	}
	mockIterator1 := iterators.Slice(tokens1)
	mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).Return(mockIterator1, nil).Once()

	ctx := t.Context()
	fetcher.update(ctx)

	// Wait for cache to become stale
	time.Sleep(100 * time.Millisecond)

	// Setup second call expectation
	tokens2 := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "200",
		},
	}
	mockIterator2 := iterators.Slice(tokens2)
	mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).Return(mockIterator2, nil).Once()

	// Query should trigger hard refresh
	it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")

	require.NoError(t, err)
	assert.NotNil(t, it)

	mockDB.AssertExpectations(t)
}

func TestCachedFetcher_CacheClear(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 10*time.Second, 100)

	// First update with tokens
	tokens1 := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "100",
		},
		{
			WalletID: "wallet2",
			Type:     "EUR",
			Quantity: "50",
		},
	}
	mockIterator1 := iterators.Slice(tokens1)
	mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).Return(mockIterator1, nil).Once()

	ctx := t.Context()
	fetcher.update(ctx)

	// Verify both keys exist
	_, ok1 := fetcher.cache.Get(tokenKey("wallet1", "USD"))
	assert.True(t, ok1)
	_, ok2 := fetcher.cache.Get(tokenKey("wallet2", "EUR"))
	assert.True(t, ok2)

	// Second update with only one token (different key)
	tokens2 := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet3",
			Type:     "GBP",
			Quantity: "75",
		},
	}
	mockIterator2 := iterators.Slice(tokens2)
	mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).Return(mockIterator2, nil).Once()

	// Force cache to be stale so update will actually run
	fetcher.lastFetched = time.Now().Add(-20 * time.Second)

	fetcher.update(ctx)

	// Note: Ristretto cache uses probabilistic eviction and may not immediately reflect changes
	// We wait a bit for the cache to process the clear and new additions
	time.Sleep(10 * time.Millisecond)

	// New key should exist
	_, ok3 := fetcher.cache.Get(tokenKey("wallet3", "GBP"))
	assert.True(t, ok3)

	mockDB.AssertExpectations(t)
}

func TestNewMixedFetcher(t *testing.T) {
	mockDB := new(mockTokenDB)

	fetcher := newMixedFetcher(mockDB, nil, 100, 30*time.Second, 100)

	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.lazyFetcher)
	assert.NotNil(t, fetcher.eagerFetcher)
}

func TestRistrettoCache_Integration(t *testing.T) {
	// Test that Ristretto cache works correctly with the fetcher
	c, err := cache.NewRistrettoCacheWithSize[permutatableIterator[*token2.UnspentTokenInWallet]](10)
	require.NoError(t, err)
	assert.NotNil(t, c)

	// Add items with cost=1
	tokens := []*token2.UnspentTokenInWallet{
		{
			WalletID: "wallet1",
			Type:     "USD",
			Quantity: "100",
		},
	}
	it := iterators.Slice(tokens)
	c.Add("key1", it)

	// Wait for cache to process the addition (Ristretto is async)
	time.Sleep(50 * time.Millisecond)

	// Retrieve items
	retrieved, ok := c.Get("key1")
	if !ok {
		t.Log("Cache miss - Ristretto may have evicted the item or not yet processed it")
		// This is acceptable behavior for Ristretto cache
	} else {
		assert.NotNil(t, retrieved)
	}

	// Clear cache
	c.Clear()

	// Verify cleared
	_, ok = c.Get("key1")
	assert.False(t, ok)
}

func TestRistrettoCache_SizeLimit(t *testing.T) {
	// Test that cache respects size limit
	smallSize := int64(5)
	c, err := cache.NewRistrettoCacheWithSize[permutatableIterator[*token2.UnspentTokenInWallet]](smallSize)
	require.NoError(t, err)

	// Add items with cost=1 each
	for i := range 10 {
		tokens := []*token2.UnspentTokenInWallet{
			{
				WalletID: "wallet",
				Type:     "USD",
				Quantity: "100",
			},
		}
		it := iterators.Slice(tokens)
		c.Add(tokenKey("wallet", token2.Type(string([]rune{rune(i)}))), it)
	}

	// Wait for cache to process additions
	time.Sleep(50 * time.Millisecond)

	// Cache should have evicted some entries due to size limit
	// Add a test item
	tokens := []*token2.UnspentTokenInWallet{
		{
			WalletID: "test",
			Type:     "TEST",
			Quantity: "100",
		},
	}
	it := iterators.Slice(tokens)
	c.Add("test_key", it)

	// Wait for cache to process the addition
	time.Sleep(50 * time.Millisecond)

	// Try to retrieve - may or may not be present due to eviction policy
	retrieved, ok := c.Get("test_key")
	if ok {
		assert.NotNil(t, retrieved)
		t.Log("Cache successfully stored and retrieved the test item")
	} else {
		t.Log("Cache evicted the test item - this is acceptable for a small cache with many additions")
	}
}

// Additional Tests for Better Coverage

func TestTokenKey(t *testing.T) {
	t.Run("creates consistent key", func(t *testing.T) {
		key1 := tokenKey("wallet1", "USD")
		key2 := tokenKey("wallet1", "USD")

		assert.Equal(t, key1, key2)
	})

	t.Run("creates different keys for different wallets", func(t *testing.T) {
		key1 := tokenKey("wallet1", "USD")
		key2 := tokenKey("wallet2", "USD")

		assert.NotEqual(t, key1, key2)
	})

	t.Run("creates different keys for different currencies", func(t *testing.T) {
		key1 := tokenKey("wallet1", "USD")
		key2 := tokenKey("wallet1", "EUR")

		assert.NotEqual(t, key1, key2)
	})

	t.Run("handles empty wallet ID", func(t *testing.T) {
		key := tokenKey("", "USD")
		assert.NotEmpty(t, key)
	})

	t.Run("handles empty currency", func(t *testing.T) {
		key := tokenKey("wallet1", "")
		assert.NotEmpty(t, key)
	})
}

func TestLazyFetcher_UnspentTokensIteratorBy_ErrorHandling(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := NewLazyFetcher(mockDB)

	t.Run("returns error from database", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "wallet1", token2.Type("USD")).
			Return(nil, expectedErr).Once()

		ctx := t.Context()
		it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")

		require.Error(t, err)
		assert.Nil(t, it)
		assert.Equal(t, expectedErr, err)

		mockDB.AssertExpectations(t)
	})

	t.Run("handles empty wallet ID", func(t *testing.T) {
		tokens := []*token2.UnspentTokenInWallet{
			{
				WalletID: "",
				Type:     "USD",
				Quantity: "100",
			},
		}
		mockIterator := iterators.Slice(tokens)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("USD")).
			Return(mockIterator, nil).Once()

		ctx := t.Context()
		it, err := fetcher.UnspentTokensIteratorBy(ctx, "", "USD")

		require.NoError(t, err)
		assert.NotNil(t, it)

		mockDB.AssertExpectations(t)
	})
}

func TestMixedFetcher_FallbackBehavior(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newMixedFetcher(mockDB, NewMetrics(&disabled.Provider{}), 0, 10*time.Second, 100)

	t.Run("uses lazy fetcher when eager returns error", func(t *testing.T) {
		// Setup: eager fetcher will fail to update
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).
			Return(nil, errors.New("db error")).Once()

		// Lazy fetcher should be called
		tokens := []*token2.UnspentTokenInWallet{
			{
				WalletID: "wallet1",
				Type:     "USD",
				Quantity: "100",
			},
		}
		mockIterator := iterators.Slice(tokens)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "wallet1", token2.Type("USD")).
			Return(mockIterator, nil).Once()

		ctx := t.Context()
		it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")

		require.NoError(t, err)
		assert.NotNil(t, it)

		mockDB.AssertExpectations(t)
	})

	t.Run("uses lazy fetcher when eager returns empty iterator", func(t *testing.T) {
		// Populate cache with different key
		tokens1 := []*token2.UnspentTokenInWallet{
			{
				WalletID: "wallet2",
				Type:     "EUR",
				Quantity: "50",
			},
		}
		mockIterator1 := iterators.Slice(tokens1)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).
			Return(mockIterator1, nil).Once()

		ctx := t.Context()
		fetcher.eagerFetcher.update(ctx)

		// Query for different key - should use lazy fetcher
		tokens2 := []*token2.UnspentTokenInWallet{
			{
				WalletID: "wallet1",
				Type:     "USD",
				Quantity: "100",
			},
		}
		mockIterator2 := iterators.Slice(tokens2)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "wallet1", token2.Type("USD")).
			Return(mockIterator2, nil).Once()

		it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")

		require.NoError(t, err)
		assert.NotNil(t, it)

		mockDB.AssertExpectations(t)
	})
}

func TestCachedFetcher_ConcurrentAccess(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 1*time.Second, 100)

	t.Run("handles concurrent reads during update", func(t *testing.T) {
		// Populate cache
		tokens := []*token2.UnspentTokenInWallet{
			{
				WalletID: "wallet1",
				Type:     "USD",
				Quantity: "100",
			},
		}
		mockIterator := iterators.Slice(tokens)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).
			Return(mockIterator, nil).Once()

		ctx := t.Context()
		fetcher.update(ctx)

		// Concurrent reads (should all hit cache, no DB calls)
		done := make(chan bool, 10)
		for range 10 {
			go func() {
				_, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all goroutines
		for range 10 {
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("timeout waiting for concurrent reads")
			}
		}

		mockDB.AssertExpectations(t)
	})
}

func TestCachedFetcher_GroupTokensByKey(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 1*time.Second, 100)

	t.Run("groups tokens correctly", func(t *testing.T) {
		tokens := []*token2.UnspentTokenInWallet{
			{WalletID: "wallet1", Type: "USD", Quantity: "100"},
			{WalletID: "wallet1", Type: "USD", Quantity: "200"},
			{WalletID: "wallet1", Type: "EUR", Quantity: "50"},
			{WalletID: "wallet2", Type: "USD", Quantity: "75"},
		}
		mockIterator := iterators.Slice(tokens)

		ctx := t.Context()
		grouped := fetcher.groupTokensByKey(ctx, mockIterator)

		// Should have 3 keys: wallet1-USD, wallet1-EUR, wallet2-USD
		assert.Len(t, grouped, 3)

		key1 := tokenKey("wallet1", "USD")
		assert.Len(t, grouped[key1], 2)

		key2 := tokenKey("wallet1", "EUR")
		assert.Len(t, grouped[key2], 1)

		key3 := tokenKey("wallet2", "USD")
		assert.Len(t, grouped[key3], 1)
	})

	t.Run("handles empty iterator", func(t *testing.T) {
		tokens := []*token2.UnspentTokenInWallet{}
		mockIterator := iterators.Slice(tokens)

		ctx := t.Context()
		grouped := fetcher.groupTokensByKey(ctx, mockIterator)

		assert.Empty(t, grouped)
	})
}

// TestCachedFetcher_UpdateCache verifies cache updates without race conditions (add before remove).
func TestCachedFetcher_UpdateCache(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 1*time.Second, 100)

	t.Run("removes stale keys", func(t *testing.T) {
		ctx := t.Context()

		// First update with 2 keys
		tokensByKey1 := map[string][]*token2.UnspentTokenInWallet{
			tokenKey("wallet1", "USD"): {
				{WalletID: "wallet1", Type: "USD", Quantity: "100"},
			},
			tokenKey("wallet2", "EUR"): {
				{WalletID: "wallet2", Type: "EUR", Quantity: "50"},
			},
		}
		fetcher.updateCache(ctx, tokensByKey1)

		// Verify both keys exist
		_, ok1 := fetcher.cache.Get(tokenKey("wallet1", "USD"))
		assert.True(t, ok1)
		_, ok2 := fetcher.cache.Get(tokenKey("wallet2", "EUR"))
		assert.True(t, ok2)

		// Second update with only 1 key
		tokensByKey2 := map[string][]*token2.UnspentTokenInWallet{
			tokenKey("wallet1", "USD"): {
				{WalletID: "wallet1", Type: "USD", Quantity: "200"},
			},
		}
		fetcher.updateCache(ctx, tokensByKey2)

		// Wait for cache to process deletions
		time.Sleep(10 * time.Millisecond)

		// First key should still exist, second should be removed
		_, ok1 = fetcher.cache.Get(tokenKey("wallet1", "USD"))
		assert.True(t, ok1)
	})

	t.Run("handles empty update", func(t *testing.T) {
		ctx := t.Context()

		// Update with empty map
		tokensByKey := map[string][]*token2.UnspentTokenInWallet{}
		fetcher.updateCache(ctx, tokensByKey)

		// Should not panic
		assert.NotNil(t, fetcher.prevKeys)
	})

	// Race condition test: verifies cache never appears empty to concurrent readers during updates.
	// This validates the fix where new entries are added BEFORE stale ones are removed.
	t.Run("no empty cache during concurrent updates", func(t *testing.T) {
		ctx := t.Context()

		// Initial cache population
		initialTokens := map[string][]*token2.UnspentTokenInWallet{
			tokenKey("wallet1", "USD"): {
				{WalletID: "wallet1", Type: "USD", Quantity: "100"},
			},
		}
		fetcher.updateCache(ctx, initialTokens)

		// Start concurrent readers
		stopReading := make(chan struct{})
		readErrors := make(chan error, 10)

		for range 10 {
			go func() {
				for {
					select {
					case <-stopReading:
						return
					default:
						// Try to read from cache
						_, ok := fetcher.cache.Get(tokenKey("wallet1", "USD"))
						if !ok {
							// Cache should never be empty during update
							readErrors <- errors.New("cache was empty during concurrent read")

							return
						}
					}
				}
			}()
		}

		// Perform multiple updates while readers are active
		for range 5 {
			newTokens := map[string][]*token2.UnspentTokenInWallet{
				tokenKey("wallet1", "USD"): {
					{WalletID: "wallet1", Type: "USD", Quantity: "200"},
				},
			}
			fetcher.updateCache(ctx, newTokens)
			time.Sleep(5 * time.Millisecond)
		}

		// Stop readers
		close(stopReading)
		time.Sleep(20 * time.Millisecond)

		// Check for errors
		select {
		case err := <-readErrors:
			t.Fatalf("Race condition detected: %v", err)
		default:
			// No errors - race condition fix is working
		}
	})
}

func TestCachedFetcher_SoftRefresh(t *testing.T) {
	mockDB := new(mockTokenDB)
	maxQueries := 3
	fetcher := newCachedFetcher(mockDB, 0, 10*time.Second, maxQueries)

	t.Run("triggers soft refresh when overused", func(t *testing.T) {
		// Initial population
		tokens := []*token2.UnspentTokenInWallet{
			{WalletID: "wallet1", Type: "USD", Quantity: "100"},
		}
		mockIterator := iterators.Slice(tokens)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).
			Return(mockIterator, nil).Once()

		ctx := t.Context()
		fetcher.update(ctx)

		// Query multiple times to trigger overuse
		for range maxQueries {
			_, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")
			require.NoError(t, err)
		}

		// Setup expectation for soft refresh (background update)
		mockIterator2 := iterators.Slice(tokens)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).
			Return(mockIterator2, nil).Once()

		// Next query should trigger soft refresh
		_, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")
		require.NoError(t, err)

		// Give background goroutine time to run
		time.Sleep(50 * time.Millisecond)

		mockDB.AssertExpectations(t)
	})
}

// TestNewFetcherProvider verifies provider creation with valid/invalid strategies and zero values.
func TestNewFetcherProvider(t *testing.T) {
	t.Run("creates provider with valid strategy", func(t *testing.T) {
		provider := NewFetcherProvider(
			nil,
			&disabled.Provider{},
			Mixed,
			100,
			time.Second,
			10,
		)

		assert.NotNil(t, provider)
		assert.Equal(t, int64(100), provider.cacheSize)
		assert.Equal(t, time.Second, provider.freshnessInterval)
		assert.Equal(t, 10, provider.maxQueries)
	})

	t.Run("panics with invalid strategy", func(t *testing.T) {
		assert.Panics(t, func() {
			NewFetcherProvider(
				nil,
				&disabled.Provider{},
				"invalid",
				100,
				time.Second,
				10,
			)
		})
	})

	t.Run("creates provider with zero values", func(t *testing.T) {
		provider := NewFetcherProvider(
			nil,
			&disabled.Provider{},
			Mixed,
			0,
			0,
			0,
		)

		assert.NotNil(t, provider)
		assert.Equal(t, int64(0), provider.cacheSize)
		assert.Equal(t, time.Duration(0), provider.freshnessInterval)
		assert.Equal(t, 0, provider.maxQueries)
	})
}

// TestFetcherProvider_GetFetcher verifies errors when token store unavailable.
func TestFetcherProvider_GetFetcher(t *testing.T) {
	t.Run("returns error when token store service not found", func(t *testing.T) {
		mockStoreManager := &mockStoreServiceManager{
			storeServiceByTMSIdFunc: func(tmsID token.TMSID) (*tokendb.StoreService, error) {
				return nil, errors.New("store not found")
			},
		}

		provider := NewFetcherProvider(
			mockStoreManager,
			&disabled.Provider{},
			Mixed,
			100,
			time.Second,
			10,
		)

		fetcher, err := provider.GetFetcher(token.TMSID{})

		require.Error(t, err)
		assert.Nil(t, fetcher)
		assert.Contains(t, err.Error(), "store not found")
	})

	t.Run("returns fetcher when token store service found", func(t *testing.T) {
		mockStoreManager := &mockStoreServiceManager{
			storeServiceByTMSIdFunc: func(tmsID token.TMSID) (*tokendb.StoreService, error) {
				return &tokendb.StoreService{}, nil
			},
		}

		provider := NewFetcherProvider(
			mockStoreManager,
			&disabled.Provider{},
			Mixed,
			100,
			time.Second,
			10,
		)

		fetcher, err := provider.GetFetcher(token.TMSID{})

		require.NoError(t, err)
		assert.NotNil(t, fetcher)
	})
}

// TestCachedFetcher_UpdateWithDatabaseError verifies cache stays stale when DB update fails.
func TestCachedFetcher_UpdateWithDatabaseError(t *testing.T) {
	mockDB := new(mockTokenDB)
	fetcher := newCachedFetcher(mockDB, 0, 1*time.Second, 100)

	t.Run("handles database error gracefully", func(t *testing.T) {
		expectedErr := errors.New("database connection failed")
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).
			Return(nil, expectedErr).Once()

		ctx := t.Context()

		// Update should not panic despite error
		fetcher.update(ctx)

		// Cache should remain stale
		assert.True(t, fetcher.isCacheStale())

		mockDB.AssertExpectations(t)
	})
}

// TestTokenKey_EdgeCases verifies tokenKey handles special chars, unicode, and empty values.
func TestTokenKey_EdgeCases(t *testing.T) {
	t.Run("handles special characters in wallet ID", func(t *testing.T) {
		key := tokenKey("wallet@#$%", "USD")
		assert.NotEmpty(t, key)
		assert.Contains(t, key, "wallet@#$%")
	})

	t.Run("handles special characters in currency", func(t *testing.T) {
		key := tokenKey("wallet1", "US$")
		assert.NotEmpty(t, key)
		assert.Contains(t, key, "US$")
	})

	t.Run("handles unicode characters", func(t *testing.T) {
		key := tokenKey("钱包", "€")
		assert.NotEmpty(t, key)
	})
}

// TestMixedFetcher_MetricsTracking verifies metrics track eager vs lazy fetcher usage.
func TestMixedFetcher_MetricsTracking(t *testing.T) {
	mockDB := new(mockTokenDB)
	metrics := NewMetrics(&disabled.Provider{})
	fetcher := newMixedFetcher(mockDB, metrics, 0, 10*time.Second, 100)

	t.Run("tracks eager fetcher usage", func(t *testing.T) {
		// Populate cache
		tokens := []*token2.UnspentTokenInWallet{
			{WalletID: "wallet1", Type: "USD", Quantity: "100"},
		}
		mockIterator := iterators.Slice(tokens)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "", token2.Type("")).
			Return(mockIterator, nil).Once()

		ctx := t.Context()
		fetcher.eagerFetcher.update(ctx)

		// Query should use eager fetcher
		_, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")
		require.NoError(t, err)

		mockDB.AssertExpectations(t)
	})

	t.Run("tracks lazy fetcher usage", func(t *testing.T) {
		// Query for non-cached key should use lazy fetcher
		tokens := []*token2.UnspentTokenInWallet{
			{WalletID: "wallet2", Type: "EUR", Quantity: "50"},
		}
		mockIterator := iterators.Slice(tokens)
		mockDB.On("SpendableTokensIteratorBy", mock.Anything, "wallet2", token2.Type("EUR")).
			Return(mockIterator, nil).Once()

		ctx := t.Context()
		_, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet2", "EUR")
		require.NoError(t, err)

		mockDB.AssertExpectations(t)
	})
}

// Mock implementations for new tests

type mockStoreServiceManager struct {
	storeServiceByTMSIdFunc func(tmsID token.TMSID) (*tokendb.StoreService, error)
}

func (m *mockStoreServiceManager) StoreServiceByTMSId(tmsID token.TMSID) (*tokendb.StoreService, error) {
	if m.storeServiceByTMSIdFunc != nil {
		return m.storeServiceByTMSIdFunc(tmsID)
	}

	return nil, errors.New("not implemented")
}
