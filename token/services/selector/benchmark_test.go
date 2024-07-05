/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package selector

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	inmemory2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock/inmemory"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple/inmemory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/testutils"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type WalletIDByRawIdentityFunc func(rawIdentity []byte) string

type Locker interface {
	Lock(id *token2.ID, txID string, reclaim bool) (string, error)
	UnlockIDs(id ...*token2.ID) []*token2.ID
	UnlockByTxID(txID string)
	IsLocked(id *token2.ID) bool
}

type extendedSelector struct {
	Selector token.Selector
	Lock     Locker
}

func (s *extendedSelector) Select(ownerFilter token.OwnerFilter, q, tokenType string) ([]*token2.ID, token2.Quantity, error) {
	return s.Selector.Select(ownerFilter, q, tokenType)
}

func (s *extendedSelector) Unselect(id ...*token2.ID) {
	if s.Lock != nil {
		s.Lock.UnlockIDs(id...)
	}
}

func BenchmarkSelectorSingle(b *testing.B) {
	settings := []Setting{
		{name: "sherdlock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSherdSelector, lockProvider: NewNoLocker},
		{name: "sherdlock+lock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSherdSelector, lockProvider: NewLocker},
		{name: "selector+nolock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewNoLocker},
		{name: "selector+lock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewLocker},
	}

	for _, s := range settings {
		var wg sync.WaitGroup
		setup(&s)
		b.ResetTimer()
		b.Run(s.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				ids, _, err := s.selector.Select(s.filter, testutils.SelectQuantity, testutils.TokenType)
				if err != nil {
					b.Error("unexpected error", err)
				}
				// release selected
				// note that currently we also measure unlocking (not ideal)
				wg.Add(1)
				go func(ids []*token2.ID) {
					defer wg.Done()
					s.selector.Unselect(ids...)
				}(ids)
			}
		})
		b.StopTimer()
		wg.Wait()
		cleanup(&s)
	}
}

func BenchmarkSelectorParallel(b *testing.B) {
	settings := []Setting{
		{name: "sherdlock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSherdSelector, lockProvider: NewNoLocker},
		{name: "sherdlock+lock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSherdSelector, lockProvider: NewLocker},
		{name: "sherdlock+lock+parallelism", clients: 10, tokens: 10 * testutils.NumTokensPerWallet, selectorProvider: NewSherdSelector, lockProvider: NewLocker},
		{name: "sherdlock+lock+contention", clients: 8, tokens: testutils.NumTokensPerWallet / 1000, selectorProvider: NewSherdSelector, lockProvider: NewLocker},
		{name: "selector+nolock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewNoLocker},
		{name: "selector+lock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewLocker},
	}

	for _, s := range settings {
		var wg sync.WaitGroup
		setup(&s)
		b.ResetTimer()
		b.Run(s.name, func(b *testing.B) {
			b.SetParallelism(s.clients)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					// select
					ids, _, err := s.selector.Select(s.filter, testutils.SelectQuantity, testutils.TokenType)
					if err != nil {
						b.Error("unexpected error", err)
					}
					// release selected
					// note that currently we also measure unlocking (not ideal)
					wg.Add(1)
					go func(ids []*token2.ID) {
						defer wg.Done()
						s.selector.Unselect(ids...)
					}(ids)
				}
			})
		})
		b.StopTimer()
		cleanup(&s)
		wg.Wait()
	}

}

func setup(s *Setting) {

	walletID := "wallet0"
	walletOwner := &token2.Owner{Raw: []byte(walletID)}

	walletIDByRawIdentity := func(rawIdentity []byte) string {
		return string(rawIdentity)
	}

	s.filter = &testutils.TokenFilter{
		Wallet:   walletOwner,
		WalletID: walletID,
	}

	// populate walletOwner
	qs := testutils.NewMockQueryService()
	for i := 0; i < s.tokens; i++ {
		q := token2.NewOneQuantity(testutils.TokenQuantityPrecision)
		t := &token2.UnspentToken{
			Id:       &token2.ID{TxId: strconv.Itoa(i), Index: 0},
			Owner:    walletOwner,
			Type:     testutils.TokenType,
			Quantity: q.Decimal(),
		}

		k := fmt.Sprintf("etoken.%s.%s.%s.%d", string(walletOwner.Raw), testutils.TokenType, t.Id.TxId, t.Id.Index)
		qs.Add(k, t)
	}

	// create lockCache to mimic efficient range queries
	qs.WarmupCache(walletID, testutils.TokenType)
	s.selector, s.cleanup = s.selectorProvider(qs, walletIDByRawIdentity, s.lockProvider())

	runtime.GC()
}

func cleanup(s *Setting) {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func NewSelector(qs *testutils.MockQueryService, walletIDByRawIdentity WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction) {

	qf := func() selector.QueryService {
		return qs
	}

	s, _ := selector.NewManager(lock, qf, testutils.SelectorNumRetries, testutils.SelectorTimeout, false, testutils.TokenQuantityPrecision).NewSelector(testutils.TxID)

	return &extendedSelector{
		Selector: s,
		Lock:     lock,
	}, nil
}

func NewSherdSelector(qs *testutils.MockQueryService, _ WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction) {
	return &extendedSelector{
		Selector: sherdlock.NewSherdSelector(testutils.TxID, qs, inmemory2.NewLocker(lock), testutils.TokenQuantityPrecision, sherdlock.NoBackoff),
		Lock:     nil,
	}, nil
}

type Setting struct {
	name             string
	clients          int
	tokens           int
	selectorProvider SelectorProviderFunction
	lockProvider     LockerProviderFunction
	selector         ExtendedSelector
	cleanup          CleanupFunction
	filter           token.OwnerFilter
}

type CleanupFunction func()
type SelectorProviderFunction func(qs *testutils.MockQueryService, walletIDByRawIdentity WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction)
type LockerProviderFunction func() selector.Locker

type ExtendedSelector interface {
	token.Selector
	Unselect(id ...*token2.ID)
}

type MockTokenIterator struct {
	*testutils.MockQueryService
	*testutils.NoLock
}

func NewLocker() selector.Locker {
	return inmemory.NewLocker(&testutils.MockVault{}, testutils.LockSleepTimeout, testutils.LockValidTxEvictionTimeout)
}

func NewNoLocker() selector.Locker {
	return &testutils.NoLock{}
}
