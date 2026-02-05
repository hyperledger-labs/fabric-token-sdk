/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	freshnessInterval     = 1 * time.Second
	maxQueries            = maxImmediateRetries
	noWalletTTLExpiration = 0 // Wallets never expire from cache when TTL is 0
)

type tokenFetcher interface {
	UnspentTokensIteratorBy(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error)
}

//go:generate counterfeiter -o mock/tokendb.go -fake-name TokenDB . TokenDB
type TokenDB interface {
	SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error)
}

type enhancedIterator[T any] interface {
	HasNext() bool
}

type permutatableIterator[T any] interface {
	iterators.Iterator[T]
	NewPermutation() iterators.Iterator[T]
}

type FetcherStrategy string

const (
	Lazy     = "lazy"
	Eager    = "eager"
	Mixed    = "mixed"
	Adaptive = "adaptive"
	Listener = "listener"
	Cached   = "cached"
)

type FetcherProvider interface {
	GetFetcher(tmsID token.TMSID) (tokenFetcher, error)
}

type fetchFunc func(db *tokendb.StoreService, notifier *tokendb.Notifier, m *Metrics) tokenFetcher

type fetcherProvider struct {
	tokenStoreServiceManager tokendb.StoreServiceManager
	notifierManager          tokendb.NotifierManager
	metrics                  *Metrics
	fetch                    fetchFunc
}

var fetchers = map[FetcherStrategy]fetchFunc{
	Mixed: func(db *tokendb.StoreService, notifier *tokendb.Notifier, m *Metrics) tokenFetcher {
		return newMixedFetcher(db, m)
	},
	Adaptive: func(db *tokendb.StoreService, notifier *tokendb.Notifier, m *Metrics) tokenFetcher {
		return newAdaptiveFetcher(db, freshnessInterval, maxQueries, m)
	},
}

func NewFetcherProvider(storeServiceManager tokendb.StoreServiceManager, notifierManager tokendb.NotifierManager, metricsProvider metrics.Provider, strategy FetcherStrategy) *fetcherProvider {
	fetcher, ok := fetchers[strategy]
	if !ok {
		panic("undefined fetcher strategy: " + strategy)
	}
	return &fetcherProvider{
		tokenStoreServiceManager: storeServiceManager,
		notifierManager:          notifierManager,
		metrics:                  newMetrics(metricsProvider),
		fetch:                    fetcher,
	}
}

func (p *fetcherProvider) GetFetcher(tmsID token.TMSID) (tokenFetcher, error) {
	tokenDB, err := p.tokenStoreServiceManager.StoreServiceByTMSId(tmsID)
	if err != nil {
		return nil, err
	}
	tokenNotifier, err := p.notifierManager.StoreServiceByTMSId(tmsID)
	if err != nil {
		return nil, err
	}

	return p.fetch(tokenDB, tokenNotifier, p.metrics), nil
}

// mixedFetcher combines both eager and lazy strategies
// In this example we return the eager result only the first time and all subsequent request are served by the lazy fetcher
// Other implementations can make different combinations, e.g. fresh results under a threshold (e.g. 10ms) can be served by the eager fetcher
// or listen for insert events in the database
type mixedFetcher struct {
	lazyFetcher  *lazyFetcher
	eagerFetcher *cachedFetcher
	m            *Metrics
}

func newMixedFetcher(tokenDB TokenDB, m *Metrics) *mixedFetcher {
	return &mixedFetcher{
		lazyFetcher:  NewLazyFetcher(tokenDB),
		eagerFetcher: newCachedFetcher(tokenDB, freshnessInterval, maxQueries),
		m:            m,
	}
}

func (f *mixedFetcher) UnspentTokensIteratorBy(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
	logger.DebugfContext(ctx, "call unspent tokens iterator")
	it, err := f.eagerFetcher.UnspentTokensIteratorBy(ctx, walletID, currency)
	logger.DebugfContext(ctx, "fetched eager iterator")
	if err == nil && it.(enhancedIterator[*token2.UnspentTokenInWallet]).HasNext() {
		logger.DebugfContext(ctx, "eager iterator had tokens. Returning iterator")
		f.m.UnspentTokensInvocations.With(fetcherTypeLabel, eager).Add(1)
		return it, nil
	}
	logger.DebugfContext(ctx, "eager iterator had no tokens. Returning lazy iterator")

	f.m.UnspentTokensInvocations.With(fetcherTypeLabel, lazy).Add(1)
	return f.lazyFetcher.UnspentTokensIteratorBy(ctx, walletID, currency)
}

// lazyFetcher only looks up the results when requested
type lazyFetcher struct {
	tokenDB TokenDB
}

func NewLazyFetcher(tokenDB TokenDB) *lazyFetcher {
	return &lazyFetcher{tokenDB: tokenDB}
}

func (f *lazyFetcher) UnspentTokensIteratorBy(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
	logger.DebugfContext(ctx, "Query the DB for new tokens")
	it, err := f.tokenDB.SpendableTokensIteratorBy(ctx, walletID, currency)
	if err != nil {
		return nil, err
	}
	return collections.NewPermutatedIterator[token2.UnspentTokenInWallet](it)
}

// cachedFetcher eagerly fetches all the tokens from the DB at regular intervals and returns the cached result
type cachedFetcher struct {
	tokenDB TokenDB
	cache   map[string]permutatableIterator[*token2.UnspentTokenInWallet]
	// freshnessInterval is the time between periodical updates
	freshnessInterval time.Duration
	// maxQueriesBeforeRefresh is the number of times the fetcher will respond with the cached result before refreshing.
	maxQueriesBeforeRefresh uint32

	// TODO: A better strategy is to keep following variables per cache key (type/owner combination) and lock/fetch only the 'expired' entry
	lastFetched      time.Time
	queriesResponded uint32
	mu               sync.RWMutex
}

func newCachedFetcher(tokenDB TokenDB, freshnessInterval time.Duration, maxQueriesBeforeRefresh int) *cachedFetcher {
	return &cachedFetcher{
		tokenDB:                 tokenDB,
		cache:                   make(map[string]permutatableIterator[*token2.UnspentTokenInWallet]),
		freshnessInterval:       freshnessInterval,
		maxQueriesBeforeRefresh: uint32(maxQueriesBeforeRefresh),
	}
}

func (f *cachedFetcher) update(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.isCacheStale() && !f.isCacheOverused() {
		logger.DebugfContext(ctx, "Cache renewed in the meantime by another process")
		return
	}
	logger.DebugfContext(ctx, "Renew token cache")
	it, err := f.tokenDB.SpendableTokensIteratorBy(ctx, "", "")
	if err != nil {
		logger.Warnf("Failed to get token iterator: %v", err)
		return
	}
	defer it.Close()

	m := map[string][]*token2.UnspentTokenInWallet{}
	for t, err := it.Next(); err == nil && t != nil; t, err = it.Next() {
		key := tokenKey(t.WalletID, t.Type)
		logger.DebugfContext(ctx, "Adding token with key [%s]", key)
		m[key] = append(m[key], t)
	}
	its := map[string]permutatableIterator[*token2.UnspentTokenInWallet]{}
	for key, toks := range m {
		its[key] = iterators.Slice(toks)
	}

	f.cache = its
	f.lastFetched = time.Now()
	atomic.StoreUint32(&f.queriesResponded, 0)
}

func (f *cachedFetcher) UnspentTokensIteratorBy(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
	defer atomic.AddUint32(&f.queriesResponded, 1)
	if f.isCacheOverused() {
		logger.DebugfContext(ctx, "Overused data. Soft refresh (in the background)...")
		go f.update(context.Background())
	}
	f.mu.RLock()
	if f.isCacheStale() {
		f.mu.RUnlock()
		logger.DebugfContext(ctx, "Stale data. Hard refresh (now)...")
		f.update(ctx)
		f.mu.RLock()
	}

	it, ok := f.cache[tokenKey(walletID, currency)]
	f.mu.RUnlock()
	if ok {
		return it.NewPermutation(), nil
	}
	logger.DebugfContext(ctx, "No tokens found in cache for [%s]. Only [%s] available. Returning empty iterator.", tokenKey(walletID, currency), collections.Keys(f.cache))
	return collections.NewEmptyIterator[*token2.UnspentTokenInWallet](), nil
}

func (f *cachedFetcher) isCacheOverused() bool {
	return atomic.LoadUint32(&f.queriesResponded) >= f.maxQueriesBeforeRefresh
}

func (f *cachedFetcher) isCacheStale() bool {
	return time.Since(f.lastFetched) > f.freshnessInterval
}

// walletInfo tracks metadata about a known wallet
type walletInfo struct {
	lastAccess time.Time // Last time this wallet was accessed
}

// adaptiveFetcher implements lazy discovery + eager refresh for known wallets only.
// It combines the benefits of both strategies: fast discovery and efficient refresh.
//
// Key Features:
// - Lazy Discovery: New wallets are loaded on-demand from the database
// - Eager Refresh: Known wallets are refreshed periodically to keep cache fresh
// - Wallet TTL: Inactive wallets can be automatically removed from cache to save memory
// - Thread-Safe: All operations are protected by sync.RWMutex
//
// Cache Refresh Triggers:
// 1. Time-based: When freshnessInterval has elapsed since last refresh
// 2. Usage-based: After maxQueriesBeforeRefresh queries have been served
//
// Wallet Expiration:
// - Wallets are tracked with their last access time
// - If walletTTL > 0, wallets not accessed within TTL are removed during refresh
// - If walletTTL = 0 (noWalletTTLExpiration), wallets never expire
type adaptiveFetcher struct {
	tokenDB TokenDB
	cache   map[string]permutatableIterator[*token2.UnspentTokenInWallet]
	// knownWallets tracks which wallets have been requested at least once and their last access time
	knownWallets map[string]*walletInfo

	// freshnessInterval is the time between periodical updates
	freshnessInterval time.Duration
	// maxQueriesBeforeRefresh is the number of times the fetcher will respond with the cached result before refreshing
	maxQueriesBeforeRefresh uint32
	// walletTTL is the maximum time a wallet can stay in cache without being accessed.
	// If 0 (noWalletTTLExpiration), wallets never expire from cache.
	// If > 0, wallets not accessed within this duration are removed during cache refresh.
	walletTTL time.Duration

	lastFetched      time.Time
	queriesResponded uint32
	mu               sync.RWMutex
	m                *Metrics
}

// newAdaptiveFetcher creates an adaptive fetcher with default settings.
// Wallets never expire from cache (walletTTL = noWalletTTLExpiration).
func newAdaptiveFetcher(tokenDB TokenDB, freshnessInterval time.Duration, maxQueriesBeforeRefresh int, m *Metrics) *adaptiveFetcher {
	return newAdaptiveFetcherWithTTL(tokenDB, freshnessInterval, maxQueriesBeforeRefresh, noWalletTTLExpiration, m)
}

// newAdaptiveFetcherWithTTL creates an adaptive fetcher with custom wallet TTL.
//
// Parameters:
//   - tokenDB: Database interface for fetching tokens
//   - freshnessInterval: Time between cache refreshes
//   - maxQueriesBeforeRefresh: Number of queries before triggering background refresh
//   - walletTTL: Time-to-live for inactive wallets
//   - 0 (noWalletTTLExpiration): Wallets never expire (default)
//   - > 0: Wallets not accessed within this duration are removed during refresh
//   - m: Metrics for tracking cache performance
//
// Example:
//
//	// No expiration (default)
//	fetcher := newAdaptiveFetcher(db, 1*time.Second, 100, metrics)
//
//	// 10-minute TTL
//	fetcher := newAdaptiveFetcherWithTTL(db, 1*time.Second, 100, 10*time.Minute, metrics)
func newAdaptiveFetcherWithTTL(tokenDB TokenDB, freshnessInterval time.Duration, maxQueriesBeforeRefresh int, walletTTL time.Duration, m *Metrics) *adaptiveFetcher {
	return &adaptiveFetcher{
		tokenDB:                 tokenDB,
		cache:                   make(map[string]permutatableIterator[*token2.UnspentTokenInWallet]),
		knownWallets:            make(map[string]*walletInfo),
		freshnessInterval:       freshnessInterval,
		maxQueriesBeforeRefresh: uint32(maxQueriesBeforeRefresh),
		walletTTL:               walletTTL,
		m:                       m,
	}
}

func (f *adaptiveFetcher) UnspentTokensIteratorBy(ctx context.Context, walletID string, currency token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
	defer atomic.AddUint32(&f.queriesResponded, 1)

	// Update last access time for this wallet
	f.mu.Lock()
	if info, exists := f.knownWallets[walletID]; exists {
		info.lastAccess = time.Now()
	} else {
		f.knownWallets[walletID] = &walletInfo{lastAccess: time.Now()}
	}
	f.mu.Unlock()

	// Trigger soft refresh if cache is overused
	if f.isCacheOverused() {
		logger.DebugfContext(ctx, "Overused data. Soft refresh (in the background)...")
		go f.update(context.Background())
	}

	key := tokenKey(walletID, currency)

	// Check cache first
	f.mu.RLock()
	needsHardRefresh := f.isCacheStale()
	it, exists := f.cache[key]
	f.mu.RUnlock()

	// Hard refresh if cache is stale
	if needsHardRefresh {
		logger.DebugfContext(ctx, "Stale data. Hard refresh (now)...")
		f.update(ctx)
		f.mu.RLock()
		it, exists = f.cache[key]
		f.mu.RUnlock()
	}

	// Cache hit - return cached result
	if exists {
		logger.DebugfContext(ctx, "Cache hit for [%s]", key)
		return it.NewPermutation(), nil
	}

	// Cache miss - lazy load this specific wallet/currency
	logger.DebugfContext(ctx, "Cache miss for [%s]. Lazy loading from database...", key)
	freshIt, err := f.tokenDB.SpendableTokensIteratorBy(ctx, walletID, currency)
	if err != nil {
		return nil, err
	}

	// Convert to slice and create permutable iterator
	tokens := []*token2.UnspentTokenInWallet{}
	for t, err := freshIt.Next(); err == nil && t != nil; t, err = freshIt.Next() {
		tokens = append(tokens, t)
	}
	freshIt.Close()

	// Add to cache and mark wallet as known (already updated in the beginning of the function)
	f.mu.Lock()
	f.cache[key] = iterators.Slice(tokens)
	logger.DebugfContext(ctx, "Added wallet [%s] to known wallets. Total known: %d", walletID, len(f.knownWallets))
	f.mu.Unlock()

	return iterators.Slice(tokens).NewPermutation(), nil
}

// update refreshes the cache for all known wallets.
// This method implements the wallet TTL expiration logic.
//
// Expiration Logic:
// - If walletTTL = 0 (noWalletTTLExpiration): All wallets are refreshed, none expire
// - If walletTTL > 0: Wallets not accessed within TTL are removed from cache
//
// The check "f.walletTTL > 0" ensures that when TTL is 0, the expiration
// condition is never evaluated, effectively disabling wallet expiration.
func (f *adaptiveFetcher) update(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.isCacheStale() && !f.isCacheOverused() {
		logger.DebugfContext(ctx, "Cache renewed in the meantime by another process")
		return
	}

	logger.DebugfContext(ctx, "Refreshing cache for %d known wallets", len(f.knownWallets))

	// Build new cache by refreshing only known wallets
	newCache := make(map[string]permutatableIterator[*token2.UnspentTokenInWallet])
	now := time.Now()
	expiredWallets := []string{}

	for walletID, info := range f.knownWallets {
		// Check if wallet has expired based on TTL
		// Note: When walletTTL = 0 (noWalletTTLExpiration), the condition "f.walletTTL > 0" is false,
		// so the expiration check is skipped and wallets never expire.
		if f.walletTTL > 0 && now.Sub(info.lastAccess) > f.walletTTL {
			logger.DebugfContext(ctx, "Wallet [%s] expired (last access: %v, TTL: %v). Removing from cache.", walletID, info.lastAccess, f.walletTTL)
			expiredWallets = append(expiredWallets, walletID)
			continue
		}

		// Query ALL token types for this wallet
		it, err := f.tokenDB.SpendableTokensIteratorBy(ctx, walletID, "")
		if err != nil {
			logger.Warnf("Failed to refresh wallet %s: %v", walletID, err)
			continue
		}

		// Group tokens by currency type for this wallet
		walletTokens := make(map[string][]*token2.UnspentTokenInWallet)
		for t, err := it.Next(); err == nil && t != nil; t, err = it.Next() {
			key := tokenKey(t.WalletID, t.Type)
			walletTokens[key] = append(walletTokens[key], t)
			logger.DebugfContext(ctx, "Adding token with key [%s]", key)
		}
		it.Close()

		// Add all currency types for this wallet to new cache
		for key, tokens := range walletTokens {
			newCache[key] = iterators.Slice(tokens)
		}
	}

	// Remove expired wallets from known wallets
	for _, walletID := range expiredWallets {
		delete(f.knownWallets, walletID)
	}

	f.cache = newCache
	f.lastFetched = time.Now()
	atomic.StoreUint32(&f.queriesResponded, 0)
	logger.DebugfContext(ctx, "Cache refreshed. Total cache entries: %d", len(f.cache))
}

func (f *adaptiveFetcher) isCacheOverused() bool {
	return atomic.LoadUint32(&f.queriesResponded) >= f.maxQueriesBeforeRefresh
}

func (f *adaptiveFetcher) isCacheStale() bool {
	return time.Since(f.lastFetched) > f.freshnessInterval
}
