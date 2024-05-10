/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	SelectorNumRetries = 2
	LockSleepTimeout   = 5
)

type Unlocker struct {
	Manager *Manager
}

func (m *Unlocker) Lock(id *token2.ID, txID string, reclaim bool) (string, error) {
	return "", nil
}

func (m *Unlocker) UnlockIDs(ids ...*token2.ID) []*token2.ID {
	return m.Manager.UnlockIDs(ids...)
}

func (m *Unlocker) UnlockByTxID(txID string) {

}

func (m *Unlocker) IsLocked(id *token2.ID) bool {
	return false
}

type QueryService interface {
	GetStatus(txID string) (token.TxStatus, string, error)
	UnspentTokensIterator() (*token.UnspentTokensIterator, error)
	UnspentTokensIteratorBy(id, typ string) (driver.UnspentTokensIterator, error)
	GetTokens(inputs ...*token2.ID) ([]*token2.Token, error)
}

type unspentTokenIterator struct {
	*UnspentTokenIterator

	seen      utils.Set[token2.ID]
	walletID  string
	tokenType string
}

func newUnspentTokenIterator(qs QueryService, mailman *Mailman, walletID, tokenType string) *token.UnspentTokensIterator {
	return &token.UnspentTokensIterator{UnspentTokensIterator: &unspentTokenIterator{
		UnspentTokenIterator: &UnspentTokenIterator{
			qs:      qs,
			mailman: mailman,
		},
		seen:      utils.NewSet[token2.ID](),
		walletID:  walletID,
		tokenType: tokenType,
	}}
}

func (m *unspentTokenIterator) Next() (*token2.UnspentToken, error) {
	res, err := m.UnspentTokenIterator.Next()
	if err != nil {
		return nil, err
	}
	if res != nil {
		m.seen[*res.Id] = struct{}{} // TODO: Add add method
		return res, nil
	}
	logger.Debugf("No more tokens found in mailman. Attempting to look up in the tokenDB...")
	// Reload from tokenDB
	it, err := m.qs.UnspentTokensIteratorBy(m.walletID, m.tokenType)
	if err != nil {
		return nil, err
	}
	updates := make([]update, 0)
	for tok, err := it.Next(); tok != nil && err == nil; tok, err = it.Next() {
		if !m.seen.Contains(*tok.Id) {
			logger.Debugf("Found token %s, will add it to mailman", tok)
			updates = append(updates, update{op: Add, tokenID: *tok.Id})
		} else {
			logger.Debugf("Found token %s, but we have already used it", tok)
		}
	}
	if len(updates) == 0 {
		logger.Debugf("No new tokens found. Returning empty result")
		return nil, nil
	}

	logger.Debugf("Updating the mailman with %d new tokens", len(updates))
	m.mailman.Update(updates...)
	for {
		if tok, err := m.UnspentTokenIterator.Next(); tok != nil || err != nil {
			return tok, err
		}
		logger.Debugf("Attempt to poll the mailman again until the new tokens have been added")
		time.Sleep(100 * time.Millisecond)
	}
}

//func newUnspentTokenIterator(qs QueryService, mailman *Mailman) *token.UnspentTokensIterator {
//	return &token.UnspentTokensIterator{UnspentTokensIterator: &UnspentTokenIterator{
//		qs:      qs,
//		mailman: mailman,
//	}}
//}

type UnspentTokenIterator struct {
	qs      QueryService
	mailman *Mailman
}

func (m *UnspentTokenIterator) Close() {
	// let's do nothing here
}

func (m *UnspentTokenIterator) Next() (*token2.UnspentToken, error) {
	return m.next()
}

func (m *UnspentTokenIterator) next() (*token2.UnspentToken, error) {
	req := &query{responseChanel: make(chan queryResponse, 1)}
	m.mailman.Query(req)

	res := <-req.responseChanel
	if res.err != nil {
		return nil, res.err
	}

	// check if tokenID empty
	if res.tokenID.Index == 0 && res.tokenID.TxId == "" {
		// no token anymore
		return nil, nil
	}

	// let's get the token
	ut := m.getUnspentToken(&res.tokenID)
	if ut == nil {
		return nil, fmt.Errorf("token not found")
	}

	return ut, nil
}

func (m *UnspentTokenIterator) getUnspentToken(tokenID *token2.ID) *token2.UnspentToken {
	tokens, err := m.qs.GetTokens(tokenID)
	if err != nil || len(tokens) != 1 {
		// token with given tokenID not found
		return nil
	}

	t := tokens[0]

	return &token2.UnspentToken{
		Id: &token2.ID{
			TxId:  tokenID.TxId,
			Index: tokenID.Index,
		},
		Owner:    t.Owner,
		Type:     t.Type,
		Quantity: t.Quantity,
	}
}
