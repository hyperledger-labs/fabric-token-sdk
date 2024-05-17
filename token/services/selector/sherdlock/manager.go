/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/common"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// LockDB enforces that a token be used only by one process
// A housekeeping job can clean up expired locks (e.g. created_at is more than 5 minutes ago) in order to:
// - avoid that the table grows infinitely
// - unlock tokens that were locked by a process that exited unexpectedly
type LockDB interface {
	// Lock locks a specific token for the consumer TX
	Lock(tokenID *token2.ID, consumerTxID core.TxID) error
	// UnlockByTxID unlocks all tokens locked by the consumer TX
	UnlockByTxID(consumerTxID core.TxID) error
}

type tokenFetcher interface {
	UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.UnspentToken], error)
}

type tokenSelectorUnlocker interface {
	token.Selector
	UnlockAll() error
}

type manager struct {
	selectorCache common.LazyProvider[core.TxID, tokenSelectorUnlocker]
}

type TokenDB interface {
	UnspentTokensIteratorBy(id, tokenType string) (driver.UnspentTokensIterator, error)
}

type iterator[k any] interface {
	Next() (k, error)
}

func NewManager(tokenDB TokenDB, lockDB LockDB, precision uint64, backoff time.Duration) *manager {
	return &manager{
		selectorCache: common.NewLazyProvider(func(txID core.TxID) (tokenSelectorUnlocker, error) {
			return NewSherdSelector(txID, tokenDB, lockDB, precision, backoff), nil
		}),
	}
}

func (m *manager) NewSelector(id core.TxID) (token.Selector, error) {
	return m.selectorCache.Get(id)
}

func (m *manager) Unlock(id core.TxID) error {
	if c, ok := m.selectorCache.Delete(id); ok {
		return c.UnlockAll()
	}
	return nil
}
