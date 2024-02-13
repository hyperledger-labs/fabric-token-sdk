/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.uber.org/zap/zapcore"
)

const (
	_       int = iota
	Valid       = network.Valid   // Transaction is valid and committed
	Invalid     = network.Invalid // Transaction is invalid and has been discarded
)

type Vault interface {
	Status(id string) (network.ValidationCode, error)
}

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
	vault                  Vault

	selectorsLock sync.RWMutex
	selectors     map[string]*SimpleSelector

	sleepTimeout time.Duration
	tmsID        token.TMSID
}

type WalletIDByRawIdentityFunc func(rawIdentity []byte) string

func NewManager(tmsID token.TMSID, vault Vault, qs QueryService, walletIDByRawIdentity WalletIDByRawIdentityFunc, tracer Tracer, tokenQuantityPrecision uint64, notifier events.Subscriber) *Manager {
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
	eventOperationMap[processor.DeleteToken] = Del

	m := &Manager{
		tmsID:                  tmsID,
		notifier:               notifier,
		eventOperationMap:      eventOperationMap,
		tracer:                 tracer,
		mailmen:                mailmen,
		tokenQuantityPrecision: tokenQuantityPrecision,
		vault:                  vault,
		qs:                     qs,
		walletIDByRawIdentity:  walletIDByRawIdentity,
		selectors:              map[string]*SimpleSelector{},
		sleepTimeout:           2 * time.Second,
	}

	// register manager as event listener, if a notifier is passed
	if notifier != nil {
		// Recall that TMS-ID is embedded in the message
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

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("received new token [{%s}:%s:%s:%s:%d]", t.TMSID, t.WalletID, t.TokenType, t.TxID, t.Index)
	}

	// check TMS ID
	if !m.tmsID.Equal(t.TMSID) {
		logger.Warnf("receive an event for a different TMS [%s]!=[%s]", m.tmsID, t.TMSID)
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
	m.selectorsLock.RLock()
	selector, ok := m.selectors[txID]
	if ok {
		m.selectorsLock.RUnlock()
		return selector, nil
	}
	m.selectorsLock.RUnlock()

	m.selectorsLock.Lock()
	defer m.selectorsLock.Unlock()

	selector, ok = m.selectors[txID]
	if ok {
		return selector, nil
	}

	selector = &SimpleSelector{
		TxID:          txID,
		QuerySelector: m,
		Precision:     m.tokenQuantityPrecision,
	}
	m.selectors[txID] = selector
	return selector, nil
}

func (m *Manager) Unlock(txID string) error {
	logger.Debugf("Call unlock with txID=%s", txID)

	m.selectorsLock.RLock()
	defer m.selectorsLock.RUnlock()
	selector, ok := m.selectors[txID]
	if !ok {
		logger.Warnf("nothing found bound to tx id [%s], no need to release", txID)
		return nil
	}
	m.UnlockIDs(selector.TokenIDs...)
	delete(m.selectors, txID)

	return nil
}

func (m *Manager) UnlockIDs(tokenIDs ...*token2.ID) []*token2.ID {
	logger.Debugf("call unlock with tokenIds [%v]", tokenIDs)

	// TODO get locked tokens from context
	// TODO make this more efficient ... looking up the space for each tokenID is super expensive

	// the GetToken interface is stupid ... only unspent tokens have IDs ... so why returning just token type ....
	// Can we assume that the output returned, has the same order as the function args?
	tokens, err := m.qs.GetTokens(tokenIDs...)
	if err != nil {
		logger.Errorf("failed to find tokens [%v], cannot return them", tokenIDs)
		return nil
	}
	logger.Debugf("unlock tokens [%v]", tokens)
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
		logger.Debugf("push back [%s:%v]", k, tokenIDs[i])
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

func (m *Manager) Start() {
	go m.scan()
}

func (m *Manager) scan() {
	for {
		logger.Debugf("token collector: scan locked tokens")
		var deleteList []string
		var unlockList []*SimpleSelector
		m.selectorsLock.RLock()
		for txID, selector := range m.selectors {
			status, err := m.vault.Status(txID)
			if err != nil {
				logger.Warnf("failed getting status for tx [%s], unlocking", txID)
				unlockList = append(unlockList, selector)
				continue
			}
			switch status {
			case Valid:
				deleteList = append(deleteList, txID)
				logger.Debugf("tx [%s] locked but valid, remove", txID)
			case Invalid:
				unlockList = append(unlockList, selector)
				logger.Debugf("tx [%s] locked but invalid, unlocking", txID)
			default:
				logger.Debugf("tx [%s] locked but status is pending, skip", txID)
			}
		}
		m.selectorsLock.RUnlock()

		m.selectorsLock.Lock()
		logger.Debugf("token collector: deleting [%d] items", len(deleteList))
		for _, s := range deleteList {
			delete(m.selectors, s)
		}
		logger.Debugf("token collector: unlocking [%d] items", len(deleteList))
		for _, s := range unlockList {
			m.UnlockIDs(s.TokenIDs...)
			delete(m.selectors, s.TxID)
		}
		m.selectorsLock.Unlock()

		for {
			logger.Debugf("token collector: sleep for some time...")
			time.Sleep(m.sleepTimeout)
			m.selectorsLock.RLock()
			l := len(m.selectors)
			m.selectorsLock.RUnlock()
			if l > 0 {
				// time to do some token collection
				logger.Debugf("token collector: time to do some token collection, [%d] locked", l)
				break
			}
		}
	}
}

func spaceKey(walletID, tokenType string) string {
	return walletID + "_" + tokenType
}
