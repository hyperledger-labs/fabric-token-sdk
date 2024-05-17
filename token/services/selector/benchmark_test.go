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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/mailman"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	inmemory2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock/inmemory"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple/inmemory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/testutils"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func BenchmarkSelectorSingle(b *testing.B) {
	settings := []Setting{
		{name: "sherdlock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSherdSelector, lockProvider: NewNoLocker},
		{name: "sherdlock+lock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSherdSelector, lockProvider: NewLocker},
		{name: "simple", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSimpleSelector, lockProvider: NewNoLocker},
		{name: "simple+mailman", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSimpleSelectorWithMailman, lockProvider: NewNoLocker},
		{name: "selector+nolock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewNoLocker},
		{name: "selector+lock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewLocker},
		{name: "selector+mailman", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelectorWithMailman, lockProvider: NewNoLocker},
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
		{name: "simple", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSimpleSelector, lockProvider: NewNoLocker},
		{name: "simple+mailman", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSimpleSelectorWithMailman, lockProvider: NewNoLocker},
		{name: "selector+nolock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewNoLocker},
		{name: "selector+lock", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelector, lockProvider: NewLocker},
		{name: "selector+mailman", clients: 1, tokens: testutils.NumTokensPerWallet, selectorProvider: NewSelectorWithMailman, lockProvider: NewNoLocker},
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

func NewSelector(qs *testutils.MockQueryService, walletIDByRawIdentity mailman.WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction) {

	qf := func() selector.QueryService {
		return qs
	}

	s, _ := selector.NewManager(lock, qf, testutils.SelectorNumRetries, testutils.SelectorTimeout, false, testutils.TokenQuantityPrecision, &testutils.MockTracer{}).NewSelector(testutils.TxID)

	return &mailman.ExtendedSelector{
		Selector: s,
		Lock:     lock,
	}, nil
}

func NewSelectorWithMailman(qs *testutils.MockQueryService, walletIDByRawIdentity mailman.WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction) {
	mmManager, err := mailman.NewManager(
		token.TMSID{},
		qs,
		walletIDByRawIdentity,
		&testutils.MockTracer{},
		testutils.TokenQuantityPrecision,
		nil,
	)
	if err != nil {
		panic(err)
	}
	mmlock := &mailman.Unlocker{Manager: mmManager}

	qf := func() selector.QueryService {
		return &mailmanManagerDecorator{mmManager, qs}
	}

	s, _ := selector.NewManager(mmlock, qf, testutils.SelectorNumRetries, testutils.SelectorTimeout, false, testutils.TokenQuantityPrecision, &testutils.MockTracer{}).NewSelector(testutils.TxID)

	return &mailman.ExtendedSelector{
		Selector: s,
		Lock:     mmlock,
	}, mmManager.Shutdown
}

func NewSherdSelector(qs *testutils.MockQueryService, _ mailman.WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction) {
	return &mailman.ExtendedSelector{
		Selector: sherdlock.NewSherdSelector(testutils.TxID, qs, inmemory2.NewLocker(lock), testutils.TokenQuantityPrecision, sherdlock.NoBackoff),
		Lock:     nil,
	}, nil
}

func NewSimpleSelector(qs *testutils.MockQueryService, walletIDByRawIdentity mailman.WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction) {
	return &mailman.ExtendedSelector{
		Selector: &mailman.SimpleSelector{QuerySelector: &MockTokenIterator{qs, nil}, Precision: testutils.TokenQuantityPrecision},
		Lock:     nil,
	}, nil
}

func NewSimpleSelectorWithMailman(qs *testutils.MockQueryService, walletIDByRawIdentity mailman.WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction) {
	mmManager, err := mailman.NewManager(
		token.TMSID{},
		qs,
		walletIDByRawIdentity,
		&testutils.MockTracer{},
		testutils.TokenQuantityPrecision,
		nil,
	)
	if err != nil {
		panic(err)
	}
	mmlock := &mailman.Unlocker{Manager: mmManager}

	return &mailman.ExtendedSelector{
		Selector: &mailman.SimpleSelector{QuerySelector: mmManager, Precision: testutils.TokenQuantityPrecision},
		Lock:     mmlock,
	}, mmManager.Shutdown
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
type SelectorProviderFunction func(qs *testutils.MockQueryService, walletIDByRawIdentity mailman.WalletIDByRawIdentityFunc, lock selector.Locker) (ExtendedSelector, CleanupFunction)
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

// only used for testing with the default selector
type mailmanManagerDecorator struct {
	*mailman.Manager
	qs selector.QueryService
}

func (m *mailmanManagerDecorator) UnspentTokensIterator() (*token.UnspentTokensIterator, error) {
	panic("Don't call me")
}

func (m *mailmanManagerDecorator) GetTokens(inputs ...*token2.ID) ([]*token2.Token, error) {
	return m.qs.GetTokens(inputs...)
}
