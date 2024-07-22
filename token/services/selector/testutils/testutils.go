/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutils

import (
	"bytes"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	NumTokensPerWallet         = 1000000
	TokenQuantityPrecision     = 64
	SelectQuantity             = "100"
	TxID                       = "someTxID"
	TokenType                  = "USD"
	SelectorNumRetries         = 2
	SelectorTimeout            = 5 * time.Second
	LockSleepTimeout           = 2 * time.Second
	LockValidTxEvictionTimeout = 5 * time.Minute
)

type MockVault struct {
}

func (m *MockVault) GetStatus(txID string) (ttxdb.TxStatus, string, error) {
	return ttxdb.Pending, "", nil
}

type MockIterator struct {
	qs   *MockQueryService
	keys []string
	pos  int
}

func (m *MockIterator) Close() {
}

func (m *MockIterator) Next() (*token2.UnspentToken, error) {
	if len(m.keys) == 0 || m.pos >= len(m.keys) {
		return nil, nil
	}

	k := m.keys[m.pos]
	t := m.qs.kvs[k]
	m.pos++
	return t, nil
}

type MockQueryService struct {
	kvs      map[string]*token2.UnspentToken
	cache    map[string][]string
	tokenIDs map[token2.ID]*token2.UnspentToken
	allKeys  []string
	asTokens map[token2.ID]*token2.Token
}

func NewMockQueryService() *MockQueryService {
	return &MockQueryService{
		kvs:      make(map[string]*token2.UnspentToken, 1024),
		cache:    make(map[string][]string, 1024),
		tokenIDs: make(map[token2.ID]*token2.UnspentToken, 1024),
		asTokens: make(map[token2.ID]*token2.Token, 1024),
	}
}

func (q *MockQueryService) Add(key string, t *token2.UnspentToken) {
	q.kvs[key] = t
	q.tokenIDs[*t.Id] = t
	q.allKeys = append(q.allKeys, key)

	to := &token2.Token{
		Owner:    t.Owner,
		Type:     t.Type,
		Quantity: t.Quantity,
	}
	q.asTokens[*t.Id] = to
}

func (q *MockQueryService) WarmupCache(walletID, tokenType string) {
	//fmt.Printf("try to find by %s %s\n", walletID, tokenType)
	keys := make([]string, 0, len(q.kvs))
	for k := range q.kvs {
		// do some filtering
		if strings.Contains(k, walletID) && strings.Contains(k, tokenType) {
			//fmt.Printf("filter key=%s\n", k)
			keys = append(keys, k)
		}
	}
	q.cache[walletID] = keys
}

func (q *MockQueryService) GetUnspentToken(tokenID *token2.ID) *token2.UnspentToken {
	t, ok := q.tokenIDs[*tokenID]
	if !ok {
		return nil
	}
	return t
}

func (q *MockQueryService) GetUnspentTokens(inputs ...*token2.ID) ([]*token2.UnspentToken, error) {
	ts := make([]*token2.UnspentToken, len(inputs))
	for i, input := range inputs {
		t, ok := q.tokenIDs[*input]
		if !ok {
			return nil, errors.Errorf("cannt find token with ID=%s", input.String())
		}
		ts[i] = t
	}
	return ts, nil
}

func (q *MockQueryService) UnspentTokensIterator() (*token.UnspentTokensIterator, error) {
	return &token.UnspentTokensIterator{UnspentTokensIterator: &MockIterator{q, q.allKeys, 0}}, nil
}

func (q *MockQueryService) MinTokenInfoIteratorBy(ownerEID string, typ string) (driver.MinTokenInfoIterator, error) {
	it, err := q.UnspentTokensIteratorBy(ownerEID, typ)
	if err != nil {
		return nil, err
	}
	return collections.Map(it, func(ut *token2.UnspentToken) (*token2.MinTokenInfo, error) {
		return &token2.MinTokenInfo{
			Id:       ut.Id,
			Owner:    string(ut.Owner.Raw),
			Type:     ut.Type,
			Quantity: ut.Quantity,
		}, nil
	}), nil
}

func (q *MockQueryService) UnspentTokensIteratorBy(id, _ string) (driver.UnspentTokensIterator, error) {
	return &token.UnspentTokensIterator{UnspentTokensIterator: &MockIterator{q, q.cache[id], 0}}, nil
}

func (q *MockQueryService) GetTokens(inputs ...*token2.ID) ([]*token2.Token, error) {
	ts := make([]*token2.Token, len(inputs))
	for i, input := range inputs {
		t, ok := q.asTokens[*input]
		if !ok {
			return nil, errors.Errorf("cannt find token with ID=%s", input.String())
		}
		ts[i] = t
	}

	return ts, nil
}

func (q *MockQueryService) GetStatus(txID string) (token.TxStatus, string, error) {
	return token.Pending, "", nil
}

type NoLock struct {
}

func (n *NoLock) Lock(id *token2.ID, txID string, reclaim bool) (string, error) {
	return "", nil
}

func (n *NoLock) UnlockIDs(id ...*token2.ID) []*token2.ID {
	return id
}

func (n *NoLock) UnlockByTxID(txID string) {
}

func (n *NoLock) IsLocked(id *token2.ID) bool {
	return false
}

type TokenFilter struct {
	Wallet   *token2.Owner
	WalletID string
}

func (c *TokenFilter) ID() string {
	return c.WalletID
}

func (c *TokenFilter) ContainsToken(token *token2.UnspentToken) bool {
	return bytes.Equal(token.Owner.Raw, c.Wallet.Raw)
}
