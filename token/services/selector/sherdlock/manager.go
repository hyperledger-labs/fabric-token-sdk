/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type LockDB = driver2.TokenLockDB

type tokenFetcher interface {
	UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.UnspentToken], error)
}

type tokenSelectorUnlocker interface {
	token.Selector
	UnlockAll() error
}

type manager struct {
	selectorCache utils.LazyProvider[transaction.ID, tokenSelectorUnlocker]
}

type TokenDB interface {
	UnspentTokensIteratorBy(id, tokenType string) (driver.UnspentTokensIterator, error)
}

type iterator[k any] interface {
	Next() (k, error)
}

func NewManager(tokenDB TokenDB, lockDB LockDB, precision uint64, backoff time.Duration) *manager {
	return &manager{
		selectorCache: utils.NewLazyProvider(func(txID transaction.ID) (tokenSelectorUnlocker, error) {
			return NewSherdSelector(txID, tokenDB, lockDB, precision, backoff), nil
		}),
	}
}

func (m *manager) NewSelector(id transaction.ID) (token.Selector, error) {
	return m.selectorCache.Get(id)
}

func (m *manager) Unlock(id transaction.ID) error {
	if c, ok := m.selectorCache.Delete(id); ok {
		return c.UnlockAll()
	}
	return nil
}
