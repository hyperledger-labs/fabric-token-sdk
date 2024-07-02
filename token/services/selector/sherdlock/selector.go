/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	logging2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	// This way we avoid deadlocks, e.g. We have 2 tokens of value 10CHF each (20 CHF in total).
	// We also have two processes that both ask for 15CHF. If both of them concurrently lock one token each,
	// they will retry maxRetry times to see if the other process in the meantime unlocked the token.
	// If not, to avoid locking these tokens forever, we roll back and unlock the tokens.
	maxImmediateRetries = 5
	NoBackoff           = -1
)

var logger = flogging.MustGetLogger("token-sdk.selector.shared")

type Iterator[V any] interface {
	Next() V
}

type tokenLocker interface {
	TryLock(*token2.ID) bool
	UnlockAll() error
}

type selector struct {
	logger    logging2.Logger
	cache     iterator[*token2.UnspentToken]
	fetcher   tokenFetcher
	locker    tokenLocker
	precision uint64
}

type stubbornSelector struct {
	*selector
	// After maxImmediateRetries attempts, the procs will roll back and unlock the tokens.
	// If two procs unlock at the same time, we have a livelock.
	// To avoid it, we back off (wait) for a random interval within some limits and retry
	backoffInterval time.Duration
	// However, it might be that we don't have a livelock, but we are simply out of funds.
	// Instead of polling forever, we can abort after a certain amount of attempts.
	maxRetriesAfterBackoff int
}

func (m *stubbornSelector) Select(owner token.OwnerFilter, q, currency string) ([]*token2.ID, token2.Quantity, error) {
	for retriesAfterBackoff := 0; retriesAfterBackoff <= m.maxRetriesAfterBackoff; retriesAfterBackoff++ {
		if tokens, quantity, err := m.selector.Select(owner, q, currency); err == nil || !errors.Is(err, token.SelectorSufficientButLockedFunds) {
			return tokens, quantity, err
		}
		backoffDuration := time.Duration(rand.Int63n(int64(m.backoffInterval)))
		m.logger.Debugf("Token selection aborted, so that other procs can retry. Release tokens and backoff for %v before retrying to select. In the meantime maybe some other process releases token locks or adds tokens.", backoffDuration)
		time.Sleep(backoffDuration)
		m.logger.Debugf("Now it is our turn to retry...")
	}
	return nil, nil, errors.Wrapf(token.SelectorInsufficientFunds, "aborted too many times and no other process unlocked or added tokens")
}

func NewStubbornSelector(logger logging2.Logger, tokenDB tokenFetcher, lockDB tokenLocker, precision uint64, backoff time.Duration) *stubbornSelector {
	return &stubbornSelector{
		selector:               NewSelector(logger, tokenDB, lockDB, precision),
		backoffInterval:        backoff,
		maxRetriesAfterBackoff: 3,
	}
}

func NewSelector(logger logging2.Logger, tokenDB tokenFetcher, lockDB tokenLocker, precision uint64) *selector {
	return &selector{
		logger:    logger,
		cache:     collections.NewEmptyIterator[*token2.UnspentToken](),
		fetcher:   tokenDB,
		locker:    lockDB,
		precision: precision,
	}
}

func (m *selector) Select(owner token.OwnerFilter, q, currency string) ([]*token2.ID, token2.Quantity, error) {
	quantity, err := token2.ToQuantity(q, m.precision)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create quantity")
	}
	sum, selected, tokensLockedByOthersExist, immediateRetries := token2.NewZeroQuantity(m.precision), collections.NewSet[*token2.ID](), true, 0
	for {
		if t, err := m.cache.Next(); err != nil {
			err2 := m.locker.UnlockAll()
			return nil, nil, errors.Wrapf(err, "failed to get tokens for [%s:%s] - unlock: %v", owner.ID(), currency, err2)
		} else if t == nil {
			if !tokensLockedByOthersExist {
				return nil, nil, errors.Wrapf(
					token.SelectorInsufficientFunds,
					"insufficient funds, only [%s] tokens of type [%s] are available, but [%s] were requested and no other process has any tokens locked",
					sum.Decimal(),
					currency,
					quantity.Decimal(),
				)
			}

			if immediateRetries > maxImmediateRetries {
				m.logger.Warnf("Exceeded max number of immediate retries. Unlock tokens and abort...")
				if err := m.locker.UnlockAll(); err != nil {
					return nil, nil, errors.Wrapf(err, "exceeded number of retries: %d and unlock failed", maxImmediateRetries)
				}

				// When we loop over the tokens, we check whether a token is already locked.
				// Every time our token cache finishes, but we noted that one of the tokens we saw was used by someone,
				// we retry to fetch, in case the other process did not spend and unlocked the token meanwhile.
				// We do not unlock our tokens, yet.
				// After some retries, we unlock the tokens and return a token.SelectorInsufficientFunds error
				return nil, nil, token.SelectorSufficientButLockedFunds
			}

			m.logger.Debugf("Fetch all non-deleted tokens from the DB and refresh the token cache.")
			if m.cache, err = m.fetcher.UnspentTokensIteratorBy(owner.ID(), currency); err != nil {
				err2 := m.locker.UnlockAll()
				return nil, nil, errors.Wrapf(err, "failed to reload tokens for retry %d [%s:%s] - unlock: %v", immediateRetries, owner.ID(), currency, err2)
			}

			immediateRetries++
			tokensLockedByOthersExist = false
		} else if locked := m.locker.TryLock(t.Id); !locked {
			m.logger.Debugf("Tried to lock token [%v], but it was already locked by another process", t)
			tokensLockedByOthersExist = true
		} else {
			m.logger.Debugf("Got the lock on token [%v]", t)
			q, err := token2.ToQuantity(t.Quantity, m.precision)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "invalid token [%s] found", t.Id)
			}
			m.logger.Debugf("Found token [%s] to add: [%s:%s].", t.Id, q.Decimal(), t.Type)
			immediateRetries = 0
			sum.Add(q)
			selected.Add(t.Id)
			if sum.Cmp(quantity) >= 0 {
				return selected.ToSlice(), sum, nil
			}
		}
	}
}

func (s *selector) UnlockAll() error {
	return s.locker.UnlockAll()
}

type fetcher struct {
	TokenDB
}

func (f *fetcher) UnspentTokensIteratorBy(walletID, currency string) (iterator[*token2.UnspentToken], error) {
	it, err := f.TokenDB.UnspentTokensIteratorBy(walletID, currency)
	if err != nil {
		return nil, err
	}
	return collections.CopyIterator[token2.UnspentToken](it)
}

type locker struct {
	LockDB
	txID transaction.ID
}

func (l *locker) TryLock(tokenID *token2.ID) bool {
	return l.LockDB.Lock(tokenID, l.txID) == nil
}

func (l *locker) UnlockAll() error {
	return l.LockDB.UnlockByTxID(l.txID)
}

func NewSherdSelector(txID transaction.ID, tokenDB TokenDB, lockDB LockDB, precision uint64, backoff time.Duration) tokenSelectorUnlocker {
	logger := logger.Named(fmt.Sprintf("selector-%s", txID))
	fetcher := &fetcher{TokenDB: tokenDB}
	locker := &locker{txID: txID, LockDB: lockDB}
	if backoff < 0 {
		return NewSelector(logger, fetcher, locker, precision)
	} else {
		return NewStubbornSelector(logger, fetcher, locker, precision, backoff)
	}
}
