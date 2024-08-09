/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	lazy2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
)

type LockDB = driver2.TokenLockDB

type tokenSelectorUnlocker interface {
	token.Selector
	UnlockAll() error
}

type manager struct {
	selectorCache lazy2.Provider[transaction.ID, tokenSelectorUnlocker]
}

type iterator[k any] interface {
	Next() (k, error)
	Close()
}

func NewManager(fetcher tokenFetcher, lockDB LockDB, precision uint64, backoff time.Duration) *manager {
	return &manager{
		selectorCache: lazy2.NewProvider(func(txID transaction.ID) (tokenSelectorUnlocker, error) {
			return NewSherdSelector(txID, fetcher, lockDB, precision, backoff), nil
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
