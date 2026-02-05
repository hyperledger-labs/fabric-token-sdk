/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	lazy2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Locker interface {
	// Lock locks a specific token for the consumer TX
	Lock(ctx context.Context, tokenID *token2.ID, consumerTxID transaction.ID) error
	// UnlockByTxID unlocks all tokens locked by the consumer TX
	UnlockByTxID(ctx context.Context, consumerTxID transaction.ID) error
	// Cleanup removes the locks such that either:
	// 1. The transaction that locked that token is valid or invalid;
	// 2. The lock is too old.
	Cleanup(ctx context.Context, leaseExpiry time.Duration) error
}

type tokenSelectorUnlocker interface {
	token.Selector
	UnlockAll(ctx context.Context) error
}

type manager struct {
	selectorCache          lazy2.Provider[transaction.ID, tokenSelectorUnlocker]
	locker                 Locker
	leaseExpiry            time.Duration
	leaseCleanupTickPeriod time.Duration
}

//go:generate counterfeiter -o mock/iterator.go  -fake-name Iterator . iterator
type iterator[k any] interface {
	Next() (k, error)
	Close()
}

func NewManager(
	fetcher tokenFetcher,
	locker Locker,
	precision uint64,
	backoff time.Duration,
	maxRetriesAfterBackOff int,
	leaseExpiry time.Duration,
	leaseCleanupTickPeriod time.Duration,
) *manager {
	m := &manager{
		locker:                 locker,
		leaseExpiry:            leaseExpiry,
		leaseCleanupTickPeriod: leaseCleanupTickPeriod,
		selectorCache: lazy2.NewProvider(func(txID transaction.ID) (tokenSelectorUnlocker, error) {
			return NewSherdSelector(txID, fetcher, locker, precision, backoff, maxRetriesAfterBackOff), nil
		}),
	}
	if leaseCleanupTickPeriod > 0 && leaseExpiry > 0 {
		go m.cleaner(context.Background())
	}
	return m
}

func (m *manager) NewSelector(id transaction.ID) (token.Selector, error) {
	return m.selectorCache.Get(id)
}

func (m *manager) Unlock(ctx context.Context, id transaction.ID) error {
	return m.locker.UnlockByTxID(ctx, id)
}

func (m *manager) Close(id transaction.ID) error {
	if c, ok := m.selectorCache.Delete(id); ok {
		return c.Close()
	}
	return errors.New("selector for " + id + " not found")
}

func (m *manager) cleaner(ctx context.Context) {
	ticker := time.NewTicker(m.leaseCleanupTickPeriod)
	defer ticker.Stop()

	for range ticker.C {
		logger.DebugfContext(ctx, "release token locks older than [%s]", m.leaseExpiry)
		if err := m.locker.Cleanup(ctx, m.leaseExpiry); err != nil {
			logger.Errorf("failed to release token locks: [%s]", err)
		}
	}
}
