/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/LFDT-Panurus/panurus/token/driver"
	token2 "github.com/LFDT-Panurus/panurus/token/token"
	"github.com/stretchr/testify/require"
)

// gatedSpendableIterator blocks the first Next until release is closed, signalling on
// started once it is parked there. That models the window the cache-invalidation race
// lives in: the snapshot has been read from the store, but update has not installed it yet.
type gatedSpendableIterator struct {
	tokens  []*token2.UnspentTokenInWallet
	i       int
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (it *gatedSpendableIterator) Next() (*token2.UnspentTokenInWallet, error) {
	it.once.Do(func() {
		close(it.started)
		<-it.release
	})
	if it.i >= len(it.tokens) {
		return nil, nil
	}
	t := it.tokens[it.i]
	it.i++

	return t, nil
}

func (it *gatedSpendableIterator) Close() {}

// gatedTokenDB hands out a gated iterator for the first read and plain ones afterwards, so
// a test can park exactly one update inside the store read.
type gatedTokenDB struct {
	mu      sync.Mutex
	calls   int
	tokens  []*token2.UnspentTokenInWallet
	started chan struct{}
	release chan struct{}
}

func (db *gatedTokenDB) SpendableTokensIteratorBy(context.Context, string, token2.Type) (driver.SpendableTokensIterator, error) {
	db.mu.Lock()
	db.calls++
	first := db.calls == 1
	db.mu.Unlock()
	if first {
		return &gatedSpendableIterator{tokens: db.tokens, started: db.started, release: db.release}, nil
	}

	return &gatedSpendableIterator{tokens: db.tokens, started: make(chan struct{}), release: closedChan()}, nil
}

func (db *gatedTokenDB) HasEnoughSpendableTokens(context.Context, string, token2.Type, *big.Int) (bool, error) {
	return true, nil
}

func (db *gatedTokenDB) callCount() int {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.calls
}

func closedChan() chan struct{} {
	c := make(chan struct{})
	close(c)

	return c
}

// TestCachedFetcher_DoesNotStampSnapshotInvalidatedMidRead pins the cache-invalidation TOCTOU:
// an update whose store read began before InvalidateCache must not stamp that snapshot fresh,
// because the snapshot cannot contain whatever made the caller invalidate. Before the fix,
// update re-checked only staleness - which an invalidation satisfies - so it installed the
// pre-invalidation snapshot and marked it fresh, silently losing the refresh for a whole
// freshness interval. The snapshot is still installed (see
// TestCachedFetcher_InstallsSnapshotInvalidatedMidRead).
func TestCachedFetcher_DoesNotStampSnapshotInvalidatedMidRead(t *testing.T) {
	db := &gatedTokenDB{
		tokens: []*token2.UnspentTokenInWallet{
			{WalletID: "wallet1", Type: "USD", Quantity: "100"},
		},
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	// A long freshness interval so that only an invalidation, never the clock, can make the
	// installed snapshot stale.
	f := NewCachedFetcher(db, 0, time.Hour, 100)

	done := make(chan struct{})
	go func() {
		defer close(done)
		f.update(context.Background())
	}()

	select {
	case <-db.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the update never reached the store read")
	}

	// The selector found a stale candidate while the snapshot above was in flight.
	f.InvalidateCache()
	close(db.release)

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the update never returned")
	}

	require.True(t, f.isCacheStale(),
		"a snapshot read before the invalidation was installed and stamped fresh, so the "+
			"invalidation is lost until the freshness interval elapses")

	// And the next read actually refetches, rather than being served the discarded snapshot.
	_, err := f.UnspentTokensIteratorBy(context.Background(), "wallet1", "USD")
	require.NoError(t, err)
	require.Equal(t, 2, db.callCount(), "the invalidated cache was not refreshed on the next read")
	require.False(t, f.isCacheStale(), "the post-invalidation snapshot should be fresh")
}

// installedTokens drains the cached iterator for wallet1/USD and returns the quantities, in order.
func installedTokens(t *testing.T, f *cachedFetcher) []string {
	t.Helper()
	it, err := f.UnspentTokensIteratorBy(context.Background(), "wallet1", "USD")
	require.NoError(t, err)
	defer it.Close()
	var out []string
	for {
		tok, err := it.Next()
		require.NoError(t, err)
		if tok == nil {
			return out
		}
		out = append(out, tok.Quantity)
	}
}

// TestCachedFetcher_InstallsSnapshotInvalidatedMidRead pins the other half of the rule: a
// snapshot overtaken by an invalidation is not stamped fresh, but it IS installed. Discarding
// it kept the previous snapshot visible, and under sustained invalidations every refresh was
// overtaken, so the cache never advanced past its first snapshot: with a settling wallet the
// selector walked ~2,000 long-spent tokens per Select and failed most requests on a wallet
// that could pay. An overtaken snapshot is newer than the one it replaces, so readers must get
// it while the cache stays stale and the next read refreshes again.
func TestCachedFetcher_InstallsSnapshotInvalidatedMidRead(t *testing.T) {
	db := &gatedTokenDB{
		tokens: []*token2.UnspentTokenInWallet{
			{WalletID: "wallet1", Type: "USD", Quantity: "100"},
		},
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	f := NewCachedFetcher(db, 0, time.Hour, 100)

	done := make(chan struct{})
	go func() {
		defer close(done)
		f.update(context.Background())
	}()
	select {
	case <-db.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the update never reached the store read")
	}
	f.InvalidateCache()
	close(db.release)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the update never returned")
	}

	require.True(t, f.isCacheStale(), "an overtaken snapshot must not be stamped fresh")
	// Read the cache directly: going through UnspentTokensIteratorBy would refresh a stale cache.
	it, ok := f.cache.Get(tokenKey("wallet1", "USD"))
	require.True(t, ok, "the overtaken snapshot was discarded instead of installed")
	tok, err := it.NewPermutation().Next()
	require.NoError(t, err)
	require.NotNil(t, tok)
	require.Equal(t, "100", tok.Quantity)
}

// advancingTokenDB serves a different single token on every read, so a test can tell which
// read the cache currently holds, and runs onRead after the read starts. The hook models the
// selector invalidating while a refresh is in flight.
type advancingTokenDB struct {
	mu     sync.Mutex
	calls  int
	onRead func(call int)
}

func (db *advancingTokenDB) SpendableTokensIteratorBy(context.Context, string, token2.Type) (driver.SpendableTokensIterator, error) {
	db.mu.Lock()
	db.calls++
	call := db.calls
	hook := db.onRead
	db.mu.Unlock()
	if hook != nil {
		hook(call)
	}

	return &gatedSpendableIterator{
		tokens:  []*token2.UnspentTokenInWallet{{WalletID: "wallet1", Type: "USD", Quantity: fmt.Sprintf("read%d", call)}},
		started: make(chan struct{}),
		release: closedChan(),
	}, nil
}

func (db *advancingTokenDB) HasEnoughSpendableTokens(context.Context, string, token2.Type, *big.Int) (bool, error) {
	return true, nil
}

// TestCachedFetcher_AdvancesUnderSustainedInvalidations is the regression test for the
// starvation: every refresh is overtaken by an invalidation (as on a wallet whose Selects keep
// finding spent candidates), yet readers must still see each newer snapshot instead of the
// first one forever.
func TestCachedFetcher_AdvancesUnderSustainedInvalidations(t *testing.T) {
	db := &advancingTokenDB{}
	f := NewCachedFetcher(db, 0, time.Hour, 1000)
	db.onRead = func(int) { f.InvalidateCache() }

	seen := map[string]bool{}
	for range 6 {
		got := installedTokens(t, f)
		require.Len(t, got, 1)
		seen[got[0]] = true
	}

	require.Greater(t, len(seen), 3,
		"the cache kept serving a stale snapshot while every refresh was overtaken by an invalidation: saw %v", seen)
	require.True(t, f.isCacheStale(), "with an invalidation behind every read the cache must never be stamped fresh")
}

// TestCachedFetcher_SettlesOnceInvalidationsStop checks that the stale marker is not sticky: once
// no invalidation lands during a refresh, the snapshot is stamped fresh and reads stop hitting
// the store.
func TestCachedFetcher_SettlesOnceInvalidationsStop(t *testing.T) {
	db := &advancingTokenDB{}
	f := NewCachedFetcher(db, 0, time.Hour, 1000)
	db.onRead = func(call int) {
		if call <= 2 {
			f.InvalidateCache()
		}
	}

	for range 6 {
		installedTokens(t, f)
	}

	require.False(t, f.isCacheStale(), "the cache was never stamped fresh once the invalidations stopped")
	db.mu.Lock()
	defer db.mu.Unlock()
	require.Equal(t, 3, db.calls, "a fresh cache must serve reads without going back to the store")
}
