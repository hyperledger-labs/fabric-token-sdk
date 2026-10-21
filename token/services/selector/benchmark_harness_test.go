/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package selector

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/LFDT-Panurus/panurus/token/driver"
	"github.com/LFDT-Panurus/panurus/token/services/selector/sherdlock"
	inmemory2 "github.com/LFDT-Panurus/panurus/token/services/selector/sherdlock/inmemory"
	"github.com/LFDT-Panurus/panurus/token/services/selector/testutils"
	token2 "github.com/LFDT-Panurus/panurus/token/token"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// harnessTokens keeps these tests fast while still holding several Selects' worth of
// testutils.SelectQuantity (100 tokens of quantity 1 each).
const harnessTokens = 1000

// TestBenchHarnessSimpleUnselectReleasesLocks checks that the "selector+lock" setting of
// BenchmarkSelectorSingle actually releases what Select locked. The simple selector locks
// under ownerFilter.ID() ("wallet0"), so Unselect must unlock under the same owner: the
// in-memory locker is sharded by owner, and unlocking under any other owner is a no-op.
func TestBenchHarnessSimpleUnselectReleasesLocks(t *testing.T) {
	s := Setting{name: "selector+lock", tokens: harnessTokens, selectorProvider: NewSelector, lockProvider: NewLocker}
	setup(&s)
	defer cleanup(&s)
	es, ok := s.selector.(*extendedSelector)
	require.True(t, ok)

	ids, _, err := s.selector.Select(t.Context(), s.filter, testutils.SelectQuantity, testutils.TokenType)
	require.NoError(t, err)
	require.Len(t, ids, 100)

	s.selector.Unselect(ids...)

	var stillLocked int
	for _, id := range ids {
		if es.Lock.IsLocked(id) {
			stillLocked++
		}
	}
	assert.Zero(t, stillLocked, "Unselect left %d/%d tokens locked", stillLocked, len(ids))
}

// TestBenchHarnessSimpleLockedSetGrows shows the effect on the benchmark: if Unselect does not
// release, every Select/Unselect round leaves another 100 tokens locked, so round k scans past
// k*100 locked tokens and ns/op grows with b.N instead of measuring a steady-state Select.
func TestBenchHarnessSimpleLockedSetGrows(t *testing.T) {
	s := Setting{name: "selector+lock", tokens: harnessTokens, selectorProvider: NewSelector, lockProvider: NewLocker}
	setup(&s)
	defer cleanup(&s)
	es, ok := s.selector.(*extendedSelector)
	require.True(t, ok)

	const rounds = 5
	seen := collections.NewSet[token2.ID]()
	for range rounds {
		ids, _, err := s.selector.Select(t.Context(), s.filter, testutils.SelectQuantity, testutils.TokenType)
		require.NoError(t, err)
		for _, id := range ids {
			seen.Add(*id)
		}
		s.selector.Unselect(ids...)
	}

	var locked int
	for _, id := range seen.ToSlice() {
		if es.Lock.IsLocked(&id) {
			locked++
		}
	}
	assert.Zero(t, locked, "after %d Select/Unselect rounds %d tokens are still locked", rounds, locked)
}

// countingTokenDB counts SpendableTokensIteratorBy calls on the wrapped mock.
type countingTokenDB struct {
	*testutils.MockQueryService
	fetches atomic.Int32
}

func (c *countingTokenDB) SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token2.Type) (driver.SpendableTokensIterator, error) {
	c.fetches.Add(1)

	return c.MockQueryService.SpendableTokensIteratorBy(ctx, walletID, typ)
}

// TestBenchHarnessSherdlockFetchesPerSelect checks that the sherdlock settings exercise the
// fetcher on every Select, as production does by giving each transaction its own selector.
// A single selector reused across iterations kept its candidate iterator between calls, so the
// lazy fetcher read the store once per ~10k selects and the fetcher strategies the benchmark
// compares were never really exercised.
func TestBenchHarnessSherdlockFetchesPerSelect(t *testing.T) {
	qs := testutils.NewMockQueryService()
	for i := range harnessTokens {
		addHarnessToken(qs, i)
	}
	qs.WarmupCache(benchWalletID, testutils.TokenType)
	db := &countingTokenDB{MockQueryService: qs}
	lock := NewLocker()
	es := newSherdlockSetting(newBenchMetricsProvider(), lock, sherdlock.NoBackoff, func(*sherdlock.Metrics) (sherdlock.TokenFetcher, sherdlock.Locker) {
		return sherdlock.NewLazyFetcher(db), inmemory2.NewLocker(lock)
	})
	filter := &testutils.TokenFilter{Wallet: []byte(benchWalletID), WalletID: benchWalletID}

	const rounds = 5
	for range rounds {
		ids, _, err := es.Select(t.Context(), filter, testutils.SelectQuantity, testutils.TokenType)
		require.NoError(t, err)
		require.Len(t, ids, 100)
		es.Unselect(ids...)
		for _, id := range ids {
			require.False(t, lock.IsLocked(id), "token %s still locked after Unselect", id)
		}
	}

	assert.Equal(t, int32(rounds), db.fetches.Load(), "store must be read once per Select")
}

// TestBenchHarnessMetricsResetPerRound checks that a round's metrics report covers that round
// only. b.Run calls the benchmark function once per b.N round on the same selector, so without
// a reset each report also included every earlier round, notably the slow first fetch.
func TestBenchHarnessMetricsResetPerRound(t *testing.T) {
	s := Setting{name: "sherdlock", tokens: harnessTokens, selectorProvider: NewSherdSelector, lockProvider: NewNoLocker}
	setup(&s)
	defer cleanup(&s)
	mp := s.selector.BenchMetrics()
	require.NotNil(t, mp)

	for _, n := range []int{1, 3} {
		resetBenchMetrics(mp)
		for range n {
			ids, _, err := s.selector.Select(t.Context(), s.filter, testutils.SelectQuantity, testutils.TokenType)
			require.NoError(t, err)
			s.selector.Unselect(ids...)
		}
	}

	_, count, ok := mp.histogramAvg("selection_duration_seconds")
	require.True(t, ok)
	assert.Equal(t, 3, count, "the last round of 3 Selects must report n=3")
	assert.Equal(t, map[string]float64{"outcome=success": 3}, mp.counterByLabel("selection_outcome_total"))
}

// addHarnessToken stores token i the same way setup does for BenchmarkSelectorSingle.
func addHarnessToken(qs *testutils.MockQueryService, i int) {
	q := token2.NewOneQuantity(testutils.TokenQuantityPrecision)
	id := token2.ID{TxId: strconv.Itoa(i), Index: 0}
	qs.Add("etoken."+benchWalletID+"."+testutils.TokenType+"."+id.TxId+".0", &token2.UnspentToken{
		Id:       id,
		Owner:    []byte(benchWalletID),
		Type:     testutils.TokenType,
		Quantity: q.Decimal(),
	})
}
