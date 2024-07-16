/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	updateInterval    = 10 * time.Second
	freshnessInterval = 1 * time.Second
)

type tokenFetcher interface {
	UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.MinTokenInfo], error)
}

type TokenDB interface {
	MinTokenInfoIteratorBy(ctx context.Context, ownerEID string, typ string) (driver.MinTokenInfoIterator, error)
}

type permutatableIterator[T any] interface {
	collections.Iterator[T]
	NewPermutation() collections.Iterator[T]
}

// mixedFetcher combines both eager and lazy strategies
// In this example we return the eager result only the first time and all subsequent request are served by the lazy fetcher
// Other implementations can make different combinations, e.g. fresh results under a threshold (e.g. 10ms) can be served by the eager fetcher
// or listen for insert events in the database
type mixedFetcher struct {
	lazyFetcher  *lazyFetcher
	eagerFetcher *eagerFetcher
	m            *Metrics
}

func newMixedFetcher(tokenDB TokenDB, m *Metrics) *mixedFetcher {
	return &mixedFetcher{
		lazyFetcher:  NewLazyFetcher(tokenDB),
		eagerFetcher: newEagerFetcher(tokenDB, updateInterval),
		m:            m,
	}
}

func (f *mixedFetcher) UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.MinTokenInfo], error) {
	if time.Since(f.eagerFetcher.lastFetched) > freshnessInterval {
		f.eagerFetcher.update()
	}
	return f.eagerFetcher.UnspentTokensIteratorBy(walletID, currency)
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

// eagerFetcher eagerly fetches all the tokens from the DB at regular intervals and returns the cached result
type eagerFetcher struct {
	tokenDB     TokenDB
	ticker      *time.Ticker
	cache       map[string]permutatableIterator[*token2.MinTokenInfo]
	mu          sync.RWMutex
	lastFetched time.Time
}

func newEagerFetcher(tokenDB TokenDB, updateInterval time.Duration) *eagerFetcher {
	f := &eagerFetcher{
		tokenDB: tokenDB,
		ticker:  time.NewTicker(updateInterval),
		cache:   make(map[string]permutatableIterator[*token2.MinTokenInfo]),
	}
	f.update()
	go func() {
		for range f.ticker.C {
			f.update()
		}
	}()
	return f
}

func (f *eagerFetcher) update() {
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
	f.mu.Lock()
	f.cache = its
	f.lastFetched = time.Now()
	f.mu.Unlock()
}

func (f *eagerFetcher) UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.MinTokenInfo], error) {
	f.mu.RLock()
	defer f.mu.RLock()
	if it, ok := f.cache[tokenKey(walletID, currency)]; ok {
		return it.NewPermutation(), nil
	}
	logger.Debugf("No tokens found in cache for [%s]. Only [%s] available. Returning empty iterator.", tokenKey(walletID, currency), collections.Keys(f.cache))
	return collections.NewEmptyIterator[*token2.MinTokenInfo](), nil
}
