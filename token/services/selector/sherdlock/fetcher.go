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
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	freshnessInterval = 1 * time.Second
	maxQueries        = maxImmediateRetries
)

type tokenFetcher interface {
	UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.MinTokenInfo], error)
}

type TokenDB interface {
	MinTokenInfoIteratorBy(ctx context.Context, ownerEID string, typ string) (driver.MinTokenInfoIterator, error)
}

type enhancedIterator[T any] interface {
	HasNext() bool
}

type permutatableIterator[T any] interface {
	collections.Iterator[T]
	NewPermutation() collections.Iterator[T]
}

type FetcherStrategy string

const (
	Lazy     = "lazy"
	Eager    = "eager"
	Mixed    = "mixed"
	Listener = "listener"
	Cached   = "cached"
)

type FetcherProvider interface {
	GetFetcher(tmsID token.TMSID) (tokenFetcher, error)
}

type fetchFunc func(db *tokendb.DB, notifier *tokendb.Notifier, m *Metrics) tokenFetcher

type fetcherProvider struct {
	dbManager       *tokendb.Manager
	notifierManager *tokendb.NotifierManager
	metrics         *Metrics
	fetch           fetchFunc
}

var fetchers = map[FetcherStrategy]fetchFunc{
	Mixed: func(db *tokendb.DB, notifier *tokendb.Notifier, m *Metrics) tokenFetcher {
		return newMixedFetcher(db, notifier, m)
	},
}

func NewFetcherProvider(dbManager *tokendb.Manager, notifierManager *tokendb.NotifierManager, metricsProvider metrics.Provider, strategy FetcherStrategy) *fetcherProvider {
	fetcher, ok := fetchers[strategy]
	if !ok {
		panic("undefined fetcher strategy: " + strategy)
	}
	return &fetcherProvider{
		dbManager:       dbManager,
		notifierManager: notifierManager,
		metrics:         newMetrics(metricsProvider),
		fetch:           fetcher,
	}
}

func (p *fetcherProvider) GetFetcher(tmsID token.TMSID) (tokenFetcher, error) {
	tokenDB, err := p.dbManager.DBByTMSId(tmsID)
	if err != nil {
		return nil, err
	}
	tokenNotifier, err := p.notifierManager.DBByTMSId(tmsID)
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

func newMixedFetcher(tokenDB TokenDB, tokenNotifier driver2.TokenNotifier, m *Metrics) *mixedFetcher {
	if err := tokenNotifier.Subscribe(func(operation driver3.Operation, m map[driver3.ColumnKey]string) {
		logger.Warnf("New operation %v: [%v]", operation, m)
	}); err != nil {
		panic(err)
	}
	return &mixedFetcher{
		lazyFetcher:  NewLazyFetcher(tokenDB),
		eagerFetcher: newCachedFetcher(tokenDB, freshnessInterval, maxQueries),
		m:            m,
	}
}

func (f *mixedFetcher) UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.MinTokenInfo], error) {
	logger.Debugf("call unspent tokens iterator")
	it, err := f.eagerFetcher.UnspentTokensIteratorBy(walletID, currency)
	logger.Debugf("fetched eager iterator")
	if err == nil && it.(enhancedIterator[*token2.MinTokenInfo]).HasNext() {
		logger.Debugf("eager iterator had tokens. Returning iterator")
		f.m.UnspentTokensInvocations.With(fetcherTypeLabel, eager).Add(1)
		return it, nil
	}
	logger.Debugf("eager iterator had no tokens. Returning lazy iterator")

	f.m.UnspentTokensInvocations.With(fetcherTypeLabel, lazy).Add(1)
	return f.lazyFetcher.UnspentTokensIteratorBy(walletID, currency)
}

// lazyFetcher only looks up the results when requested
type lazyFetcher struct {
	tokenDB TokenDB
}

func NewLazyFetcher(tokenDB TokenDB) *lazyFetcher {
	return &lazyFetcher{tokenDB: tokenDB}
}

func (f *lazyFetcher) UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.MinTokenInfo], error) {
	logger.Debugf("Query the DB for new tokens")
	it, err := f.tokenDB.MinTokenInfoIteratorBy(context.TODO(), walletID, currency)
	if err != nil {
		return nil, err
	}
	return collections.NewPermutatedIterator[token2.MinTokenInfo](it)
}

// cachedFetcher eagerly fetches all the tokens from the DB at regular intervals and returns the cached result
type cachedFetcher struct {
	tokenDB TokenDB
	cache   map[string]permutatableIterator[*token2.MinTokenInfo]
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
		cache:                   make(map[string]permutatableIterator[*token2.MinTokenInfo]),
		freshnessInterval:       freshnessInterval,
		maxQueriesBeforeRefresh: uint32(maxQueriesBeforeRefresh),
	}
}

func (f *cachedFetcher) update() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.isCacheStale() && !f.isCacheOverused() {
		logger.Debugf("Cache renewed in the meantime by another process")
		return
	}
	logger.Debugf("Renew token cache")
	it, err := f.tokenDB.MinTokenInfoIteratorBy(context.TODO(), "", "")
	if err != nil {
		logger.Warnf("Failed to get token iterator: %v", err)
		return
	}
	defer it.Close()

	m := map[string][]*token2.MinTokenInfo{}
	for t, err := it.Next(); err == nil && t != nil; t, err = it.Next() {
		key := tokenKey(t.Owner, t.Type)
		logger.Debugf("Adding token with key [%s]", key)
		m[key] = append(m[key], t)
	}
	its := map[string]permutatableIterator[*token2.MinTokenInfo]{}
	for key, toks := range m {
		its[key] = collections.NewSliceIterator(toks)
	}

	f.cache = its
	f.lastFetched = time.Now()
	atomic.StoreUint32(&f.queriesResponded, 0)
}

func (f *cachedFetcher) UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.MinTokenInfo], error) {
	defer atomic.AddUint32(&f.queriesResponded, 1)
	if f.isCacheOverused() {
		logger.Debugf("Overused data. Soft refresh (in the background)...")
		go f.update()
	}
	f.mu.RLock()
	if f.isCacheStale() {
		f.mu.RUnlock()
		logger.Debugf("Stale data. Hard refresh (now)...")
		f.update()
		f.mu.RLock()
	}

	it, ok := f.cache[tokenKey(walletID, currency)]
	f.mu.RUnlock()
	if ok {
		return it.NewPermutation(), nil
	}
	logger.Debugf("No tokens found in cache for [%s]. Only [%s] available. Returning empty iterator.", tokenKey(walletID, currency), collections.Keys(f.cache))
	return collections.NewEmptyIterator[*token2.MinTokenInfo](), nil
}

func (f *cachedFetcher) isCacheOverused() bool {
	return atomic.LoadUint32(&f.queriesResponded) >= f.maxQueriesBeforeRefresh
}

func (f *cachedFetcher) isCacheStale() bool {
	return time.Since(f.lastFetched) > f.freshnessInterval
}
