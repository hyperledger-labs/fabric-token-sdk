/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"fmt"

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
