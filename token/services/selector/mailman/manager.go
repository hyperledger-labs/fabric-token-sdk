/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type WalletManager interface {
	OwnerWalletByIdentity(identity view.Identity) *token.OwnerWallet
}

type VaultProvide interface {
	NewQueryEngine() *token.QueryEngine
}

type Tracer tracing.Tracer

type Manager struct {
	notifier               events.Subscriber
	eventOperationMap      map[string]Op
	tracer                 Tracer
	mailmen                map[string]*Mailman
	tokenQuantityPrecision uint64
	qs                     QueryService
	walletIDByRawIdentity  WalletIDByRawIdentityFunc
}

type WalletIDByRawIdentityFunc func(rawIdentity []byte) string

func NewManager(qs QueryService, walletIDByRawIdentity WalletIDByRawIdentityFunc, tracer Tracer, tokenQuantityPrecision uint64, notifier events.Subscriber) *Manager {
	// pre-populate mailman instances
	iter, err := qs.UnspentTokensIterator()
	if err != nil {
		return nil
	}
	defer iter.Close()

	spaces := make(map[string][]*token2.ID)
	for {
		// go through all tokens in the vault
		ut, err := iter.Next()
		if err != nil {
			continue
		}

		if ut == nil {
			// no more tokens
			break
		}

		// calculate mailman key space
		walletID := walletIDByRawIdentity(ut.Owner.Raw)
		//walletID := walletManager.OwnerWalletByIdentity(ut.Owner.Raw).ID()
		k := spaceKey(walletID, ut.Type)

		ids, ok := spaces[k]
		if !ok {
			// new space found
			spaces[k] = []*token2.ID{ut.Id}
		} else {
			spaces[k] = append(ids, ut.Id)
		}
	}

	mailmen := make(map[string]*Mailman)
	for k, ids := range spaces {
		// create mailman instance for each space
		mm := NewMailman(ids...)
		mm.Start()
		mailmen[k] = mm
		logger.Debugf("Registered new mailman instance for k=%s", k)
	}

	// define events and mailman actions
	eventOperationMap := make(map[string]Op)
	eventOperationMap[processor.AddToken] = Add
	// TODO double check if updateToken is actually needed
	eventOperationMap[processor.UpdateToken] = Add
	eventOperationMap[processor.DeleteToken] = Del

	m := &Manager{
		notifier:               notifier,
		eventOperationMap:      eventOperationMap,
		tracer:                 tracer,
		mailmen:                mailmen,
		tokenQuantityPrecision: tokenQuantityPrecision,
		qs:                     qs,
		walletIDByRawIdentity:  walletIDByRawIdentity,
	}

	// TODO what should we do if no notifier passed?
	// register manager as event listener
	if notifier != nil {
		// TODO What about network/namespace?
		for topic := range eventOperationMap {
			notifier.Subscribe(topic, m)
		}
	}

	return m
}

func (m *Manager) OnReceive(event events.Event) {
	t, ok := event.Message().(processor.TokenMessage)
	if !ok {
		logger.Warnf("cannot cast to TokenMessage %v", event.Message())
		// drop this event
		return
	}

	// sanity check that we really registered for this type of event
	op, ok := m.eventOperationMap[event.Topic()]
	if !ok {
		logger.Warnf("receive an event we did not registered for %v", event.Message())
		// drop this event
		return
	}

	// check if event contains walletID and tokenType
	// if not, we need to fetch it from the vault
	if len(t.WalletID) == 0 || len(t.TokenType) == 0 {
		tokens, err := m.qs.GetTokens(&token2.ID{TxId: t.TxID, Index: t.Index})
		if err != nil || len(tokens) != 1 {
			// cannot fetch token from vault, drop this event
			return
		}

		walletID := m.walletIDByRawIdentity(tokens[0].Owner.Raw)
		t.WalletID = walletID
		t.TokenType = tokens[0].Type
	}

	// find mailman instance responsible for this token event
	k := spaceKey(t.WalletID, t.TokenType)
	mm, ok := m.mailmen[k]
	if !ok {
		// there is no mailman instance for this walletID/tokenType - let's create one
		logger.Debugf("Registered new mailman instance for k=%s", k)

		mm = NewMailman()
		mm.Start()
		m.mailmen[k] = mm
	}

	upd := update{op: op, tokenID: token2.ID{TxId: t.TxID, Index: t.Index}}
	mm.Update(upd)
}

func (m *Manager) UnspentTokensIteratorBy(walletId, tokenType string) (*token.UnspentTokensIterator, error) {
	k := spaceKey(walletId, tokenType)

	mm, ok := m.mailmen[k]
	if !ok {
		return nil, fmt.Errorf("no mailman for this wallet ID / token type combination? k = %s", k)
	}

	return &token.UnspentTokensIterator{UnspentTokensIterator: &UnspentTokenIterator{
		qs:      m.qs,
		mailman: mm,
	}}, nil
}

func (m *Manager) Shutdown() {
	for _, mm := range m.mailmen {
		mm.Stop()
	}
}

func (m *Manager) NewSelector(txID string) (token.Selector, error) {
	// TODO should we store txID context so we can unlock later
	// it seems that his is not needed anymore as we use UnlockIDs
	return &SimpleSelector{QuerySelector: m, Precision: m.tokenQuantityPrecision}, nil
}

func (m *Manager) Unlock(txID string) error {
	logger.Debugf("Call unlock with txID=%s", txID)

	// TODO get locked tokens from context
	// it seems that his is not needed anymore as we use UnlockIDs
	return nil
}

func (m *Manager) UnlockIDs(tokenIDs ...*token2.ID) []*token2.ID {
	logger.Debugf("Call unlock with tokenIds=%v", tokenIDs)

	// TODO get locked tokens from context
	// TODO make this more efficient ... looking up the space for each tokenID is super expensive

	// the GetToken interface is stupid ... only unspent tokens have IDs ... so why returning just token type ....
	// Can we assume that the output returned, has the same order as the function args?
	tokens, err := m.qs.GetTokens(tokenIDs...)
	if err != nil {
		return nil
	}
	spaces := make(map[string][]update, len(tokenIDs))
	for i, t := range tokens {
		walletID := m.walletIDByRawIdentity(t.Owner.Raw)
		k := spaceKey(walletID, t.Type)
		ids, ok := spaces[k]
		if !ok {
			spaces[k] = []update{{op: Add, tokenID: *tokenIDs[i]}}
		} else {
			spaces[k] = append(ids, update{op: Add, tokenID: *tokenIDs[i]})
		}
	}

	for k, ups := range spaces {
		mm, ok := m.mailmen[k]
		if !ok {
			logger.Warnf("ouch! no space=%s found", k)
			return nil
		}
		mm.Update(ups...)
	}

	return tokenIDs
}

func spaceKey(walletID, tokenType string) string {
	return walletID + "_" + tokenType
}
