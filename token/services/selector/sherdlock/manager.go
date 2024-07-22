/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Locker interface {
	// Lock locks a specific token for the consumer TX
	Lock(tokenID *token2.ID, consumerTxID transaction.ID) error
	// UnlockByTxID unlocks all tokens locked by the consumer TX
	UnlockByTxID(consumerTxID transaction.ID) error
	// Cleanup removes the locks such that either:
	// 1. The transaction that locked that token is valid or invalid;
	// 2. The lock is too old.
	Cleanup(evictionDelay time.Duration) error
}

type tokenSelectorUnlocker interface {
	token.Selector
	UnlockAll() error
}

type manager struct {
	selectorCache utils.LazyProvider[transaction.ID, tokenSelectorUnlocker]
	locker        Locker
	evictionDelay time.Duration
	cleanupPeriod time.Duration
}

type iterator[k any] interface {
	Next() (k, error)
	Close()
}

func NewManager(
	tokenDB TokenDB,
	locker Locker,
	metrics *Metrics,
	precision uint64,
	backoff time.Duration,
	evictionDelay time.Duration,
) *manager {
	fetcher := newMixedFetcher(tokenDB, metrics)
	m := &manager{
		locker:        locker,
		evictionDelay: evictionDelay,
		selectorCache: utils.NewLazyProvider(func(txID transaction.ID) (tokenSelectorUnlocker, error) {
			return NewSherdSelector(txID, fetcher, locker, precision, backoff), nil
		}),
		cleanupPeriod: cleanupPeriod,
	}
	if cleanupTickPeriod > 0 {
		go m.cleaner()
	}
	return m
}

func (m *manager) NewSelector(id transaction.ID) (token.Selector, error) {
	return m.selectorCache.Get(id)
}

func (m *manager) Unlock(id transaction.ID) error {
	return m.locker.UnlockByTxID(id)
}

func (m *manager) Close(id transaction.ID) error {
	if c, ok := m.selectorCache.Delete(id); ok {
		return c.Close()
	}
	return errors.New("selector for " + id + " not found")
}

func (m *manager) cleaner() {
	ticker := time.NewTicker(5 * time.Second) // Change the duration as needed
	defer ticker.Stop()

	for range ticker.C {
		logger.Debugf("cleanup locked tokens with eviction delay of [%s]", m.evictionDelay)
		if err := m.locker.Cleanup(m.evictionDelay); err != nil {
			logger.Errorf("failed cleaning up eviction locks: %s", err)
		}
	}
}
