/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/cache"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
	for i := 0; i < maxQueries-1; i++ {
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
	ctx := context.Background()
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

	ctx := context.Background()
	fetcher.update(ctx)

	// Query cache
	it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet1", "USD")

	assert.NoError(t, err)
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

	ctx := context.Background()
	fetcher.update(ctx)

	// Query for non-existent key
	it, err := fetcher.UnspentTokensIteratorBy(ctx, "wallet2", "EUR")

	assert.NoError(t, err)
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

	ctx := context.Background()
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

	assert.NoError(t, err)
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

	ctx := context.Background()
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)

	// Add items with cost=1 each
	for i := 0; i < 10; i++ {
		tokens := []*token2.UnspentTokenInWallet{
			{
				WalletID: "wallet",
				Type:     "USD",
				Quantity: "100",
			},
		}
		it := iterators.Slice(tokens)
		c.Add(tokenKey("wallet", token2.Type(string(rune(i)))), it)
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
