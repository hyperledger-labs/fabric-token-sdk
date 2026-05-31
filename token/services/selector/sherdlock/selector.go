/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	// This way we avoid deadlocks, e.g. We have 2 tokens of value 10CHF each (20 CHF in total).
	// We also have two processes that both ask for 15CHF. If both of them concurrently lock one token each,
	// they will retry maxRetry times to see if the other process in the meantime unlocked the token.
	// If not, to avoid locking these tokens forever, we roll back and unlock the tokens.
	maxImmediateRetries = 5
	NoBackoff           = -1
)

var logger = logging.MustGetLogger()

func Logger() logging.Logger {
	return logger
}

type Selector struct {
	logger    logging.Logger
	cache     Iterator[*token2.UnspentTokenInWallet]
	fetcher   TokenFetcher
	locker    TokenLocker
	precision uint64
	metrics   *Metrics
	mu        sync.Mutex // protects cache field for concurrent Close() calls

	// Resource limits to prevent algorithmic attacks
	maxTokensPerSelection int
	maxLockAttempts       int
	selectionTimeout      time.Duration

	// Resource tracking counters (reset per selection)
	tokensIteratedCount int
	lockAttemptsCount   int
}

type StubbornSelector struct {
	*Selector
	// After maxImmediateRetries attempts, the procs will roll back and unlock the tokens.
	// If two procs unlock at the same time, we have a livelock.
	// To avoid it, we back off (wait) for a random interval within some limits and retry
	backoffInterval time.Duration
	// However, it might be that we don't have a livelock, but we are simply out of funds.
	// Instead of polling forever, we can abort after a certain amount of attempts.
	maxRetriesAfterBackoff int
	// Maximum number of outer retry cycles (enforced at StubbornSelector level)
	maxRetryCycles int
}

func (m *StubbornSelector) Select(ctx context.Context, ownerFilter token.OwnerFilter, q string, tokenType token2.Type) ([]*token2.ID, token2.Quantity, error) {
	start := time.Now()

	// Reset resource tracking counters for this selection
	m.tokensIteratedCount = 0
	m.lockAttemptsCount = 0

	// Create timeout context if configured
	timeoutCtx, cancel := context.WithTimeout(ctx, m.selectionTimeout)
	defer cancel()

	for retriesAfterBackoff := 0; retriesAfterBackoff <= m.maxRetriesAfterBackoff; retriesAfterBackoff++ {
		// Check retry cycle limit
		if retriesAfterBackoff > m.maxRetryCycles {
			if err := m.locker.UnlockAll(ctx); err != nil {
				m.logger.Errorf("failed to unlock tokens after exceeding retry cycles: %s", err)
			}
			m.metrics.SelectionDuration.Observe(time.Since(start).Seconds())
			m.metrics.SelectionOutcome.With(outcomeLabel, "error").Add(1)

			return nil, nil, errors.Errorf(
				"token selection aborted: exceeded max retry cycles (%d) after examining %d tokens and %d lock attempts",
				m.maxRetryCycles, m.tokensIteratedCount, m.lockAttemptsCount,
			)
		}

		if tokens, quantity, err := m.selectWithoutMetrics(timeoutCtx, ownerFilter, q, tokenType); err == nil || !errors.Is(err, token.SelectorSufficientButLockedFunds) {
			m.metrics.SelectionDuration.Observe(time.Since(start).Seconds())

			// Check if we hit the timeout
			if errors.Is(err, context.DeadlineExceeded) {
				if unlockErr := m.locker.UnlockAll(ctx); unlockErr != nil {
					m.logger.Errorf("failed to unlock tokens after timeout: %s", unlockErr)
				}
				m.metrics.SelectionOutcome.With(outcomeLabel, "error").Add(1)

				return nil, nil, fmt.Errorf(
					"token selection aborted: exceeded timeout (%v) after examining %d tokens and %d lock attempts: %w",
					m.selectionTimeout, m.tokensIteratedCount, m.lockAttemptsCount, err,
				)
			}

			if err == nil {
				m.metrics.SelectionOutcome.With(outcomeLabel, "success").Add(1)
			} else if errors.Is(err, token.SelectorInsufficientFunds) {
				m.metrics.SelectionOutcome.With(outcomeLabel, "insufficient_funds").Add(1)
			} else {
				m.metrics.SelectionOutcome.With(outcomeLabel, "error").Add(1)
			}

			return tokens, quantity, err
		}
		var backoffDuration time.Duration
		if m.backoffInterval > 0 {
			backoffDuration = time.Duration(rand.Int64N(int64(m.backoffInterval)))
		}
		m.logger.DebugfContext(ctx, "Token selection aborted, so that other procs can retry. Release tokens and backoff for %v before retrying to select. In the meantime maybe some other process releases token locks or adds tokens.", backoffDuration)
		select {
		case <-time.After(backoffDuration):
		case <-timeoutCtx.Done():
			if err := m.locker.UnlockAll(ctx); err != nil {
				m.logger.Errorf("failed to unlock tokens on context cancellation: %s", err)
			}
			m.metrics.SelectionDuration.Observe(time.Since(start).Seconds())
			m.metrics.SelectionOutcome.With(outcomeLabel, "error").Add(1)

			if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
				return nil, nil, fmt.Errorf(
					"token selection aborted: exceeded timeout (%v) after examining %d tokens and %d lock attempts: %w",
					m.selectionTimeout, m.tokensIteratedCount, m.lockAttemptsCount, timeoutCtx.Err(),
				)
			}

			return nil, nil, timeoutCtx.Err()
		}
		m.logger.DebugfContext(ctx, "Now it is our turn to retry...")
	}

	m.metrics.SelectionDuration.Observe(time.Since(start).Seconds())
	m.metrics.SelectionOutcome.With(outcomeLabel, "locked_funds").Add(1)

	return nil, nil, errors.Wrapf(token.SelectorInsufficientFunds, "aborted too many times and no other process unlocked or added tokens")
}

func NewStubbornSelector(logger logging.Logger, tokenDB TokenFetcher, lockDB TokenLocker, precision uint64, backoff time.Duration, retries int, maxTokensPerSelection int, maxLockAttempts int, maxRetryCycles int, selectionTimeout time.Duration, m *Metrics) *StubbornSelector {
	return &StubbornSelector{
		Selector:               NewSelector(logger, tokenDB, lockDB, precision, maxTokensPerSelection, maxLockAttempts, selectionTimeout, m),
		backoffInterval:        backoff,
		maxRetriesAfterBackoff: retries,
		maxRetryCycles:         maxRetryCycles,
	}
}

func NewSelector(logger logging.Logger, tokenDB TokenFetcher, lockDB TokenLocker, precision uint64, maxTokensPerSelection int, maxLockAttempts int, selectionTimeout time.Duration, m *Metrics) *Selector {
	return &Selector{
		logger:                logger,
		cache:                 collections.NewEmptyIterator[*token2.UnspentTokenInWallet](),
		fetcher:               tokenDB,
		locker:                lockDB,
		precision:             precision,
		maxTokensPerSelection: maxTokensPerSelection,
		maxLockAttempts:       maxLockAttempts,
		selectionTimeout:      selectionTimeout,
		metrics:               m,
	}
}

func (s *Selector) Select(ctx context.Context, owner token.OwnerFilter, q string, tokenType token2.Type) ([]*token2.ID, token2.Quantity, error) {
	start := time.Now()

	// Reset resource tracking counters for this selection
	s.tokensIteratedCount = 0
	s.lockAttemptsCount = 0

	// Create timeout context if configured
	timeoutCtx, cancel := context.WithTimeout(ctx, s.selectionTimeout)
	defer cancel()

	ids, quantity, immediateRetries, err := s.selectInternal(timeoutCtx, owner, q, tokenType)

	// Check if we hit the timeout
	if errors.Is(err, context.DeadlineExceeded) {
		// Use original context for cleanup to ensure it completes
		if err2 := s.locker.UnlockAll(ctx); err2 != nil {
			s.logger.Warnf("failed to unlock tokens after timeout: %v", err2)
		}
		s.metrics.SelectionDuration.Observe(time.Since(start).Seconds())
		s.metrics.SelectionOutcome.With(outcomeLabel, "error").Add(1)

		return nil, nil, fmt.Errorf(
			"token selection aborted: exceeded timeout (%v) after examining %d tokens and %d lock attempts: %w",
			s.selectionTimeout, s.tokensIteratedCount, s.lockAttemptsCount, err,
		)
	}

	if err != nil {
		if err2 := s.locker.UnlockAll(ctx); err2 != nil {
			s.logger.Warnf("failed to unlock tokens after selection error: %v", err2)
		}
	}
	s.metrics.SelectionDuration.Observe(time.Since(start).Seconds())
	s.metrics.ImmediateRetries.Observe(float64(immediateRetries))
	if err == nil {
		s.metrics.SelectionOutcome.With(outcomeLabel, "success").Add(1)
	} else if errors.Is(err, token.SelectorSufficientButLockedFunds) {
		s.metrics.SelectionOutcome.With(outcomeLabel, "locked_funds").Add(1)
	} else if errors.Is(err, token.SelectorInsufficientFunds) {
		s.metrics.SelectionOutcome.With(outcomeLabel, "insufficient_funds").Add(1)
	} else {
		s.metrics.SelectionOutcome.With(outcomeLabel, "error").Add(1)
	}

	return ids, quantity, err
}

// selectWithoutMetrics is used by StubbornSelector to avoid double-counting metrics.
func (s *Selector) selectWithoutMetrics(ctx context.Context, owner token.OwnerFilter, q string, tokenType token2.Type) ([]*token2.ID, token2.Quantity, error) {
	ids, quantity, _, err := s.selectInternal(ctx, owner, q, tokenType)
	if err != nil {
		if err2 := s.locker.UnlockAll(ctx); err2 != nil {
			s.logger.Warnf("failed to unlock tokens after selection error: %v", err2)
		}
	}

	return ids, quantity, err
}

func (s *Selector) selectInternal(ctx context.Context, owner token.OwnerFilter, q string, tokenType token2.Type) ([]*token2.ID, token2.Quantity, int, error) {
	if s.isClosed() {
		return nil, nil, 0, errors.Errorf("selector is already closed")
	}
	quantity, err := token2.ToQuantity(q, s.precision)
	if err != nil {
		return nil, nil, 0, errors.Wrapf(err, "failed to create quantity")
	}
	sum, selected, tokensLockedByOthersExist, immediateRetries := token2.NewZeroQuantity(s.precision), collections.NewSet[*token2.ID](), true, 0
	for {
		// Check token iteration limit
		s.tokensIteratedCount++
		if s.tokensIteratedCount > s.maxTokensPerSelection {
			return nil, nil, immediateRetries, errors.Errorf(
				"token selection aborted: exceeded max token iteration limit (%d tokens)",
				s.maxTokensPerSelection,
			)
		}

		if t, err := s.cache.Next(); err != nil {
			return nil, nil, immediateRetries, errors.Wrapf(err, "failed to get tokens for [%s:%s]", owner.ID(), tokenType)
		} else if t == nil {
			if !tokensLockedByOthersExist {
				return nil, nil, immediateRetries, errors.Wrapf(
					token.SelectorInsufficientFunds,
					"insufficient funds, only [%s] tokens of type [%s] are available, but [%s] were requested and no other process has any tokens locked",
					sum.Decimal(),
					tokenType,
					quantity.Decimal(),
				)
			}

			if immediateRetries > maxImmediateRetries {
				s.logger.Warnf("Exceeded max number of immediate retries. Unlock tokens and abort...")

				// When we loop over the tokens, we check whether a token is already locked.
				// Every time our token cache finishes, but we noted that one of the tokens we saw was used by someone,
				// we retry to fetch, in case the other process did not spend and unlocked the token meanwhile.
				// We do not unlock our tokens, yet.
				// After some retries, we unlock the tokens and return a token.SelectorInsufficientFunds error
				return nil, nil, immediateRetries, token.SelectorSufficientButLockedFunds
			}

			s.logger.DebugfContext(ctx, "Fetch all non-deleted tokens from the DB and refresh the token cache.")
			if s.cache, err = s.fetcher.UnspentTokensIteratorBy(ctx, owner.ID(), tokenType); err != nil {
				return nil, nil, immediateRetries, errors.Wrapf(err, "failed to reload tokens for retry %d [%s:%s]", immediateRetries, owner.ID(), tokenType)
			}

			immediateRetries++
			tokensLockedByOthersExist = false
		} else {
			// Check lock attempt limit
			s.lockAttemptsCount++
			if s.lockAttemptsCount > s.maxLockAttempts {
				return nil, nil, immediateRetries, errors.Errorf(
					"token selection aborted: exceeded max lock attempts (%d) after examining %d tokens",
					s.maxLockAttempts, s.tokensIteratedCount,
				)
			}

			if locked := s.locker.TryLock(ctx, &t.Id); !locked {
				s.logger.DebugfContext(ctx, "Tried to lock token [%v], but it was already locked by another process", t)
				tokensLockedByOthersExist = true
			} else {
				s.logger.DebugfContext(ctx, "Got the lock on token [%v]", t)
				q, err := token2.ToQuantity(t.Quantity, s.precision)
				if err != nil {
					return nil, nil, immediateRetries, errors.Wrapf(err, "invalid token [%s] found", t.Id)
				}
				s.logger.DebugfContext(ctx, "Found token [%s] to add: [%s:%s].", t.Id, q.Decimal(), t.Type)
				immediateRetries = 0
				sum, err = sum.Add(q)
				if err != nil {
					return nil, nil, immediateRetries, errors.Wrapf(err, "failed to add quantity")
				}
				selected.Add(&t.Id)
				if sum.Cmp(quantity) >= 0 {
					return selected.ToSlice(), sum, immediateRetries, nil
				}
			}
		}
	}
}

func (s *Selector) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cache == nil {
		return errors.New("selector is already closed")
	}
	s.cache.Close()
	s.cache = nil

	return nil
}

func (s *Selector) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cache == nil
}

func (s *Selector) UnlockAll(ctx context.Context) error {
	return s.locker.UnlockAll(ctx)
}

func tokenKey(walletID string, typ token2.Type) string {
	return fmt.Sprintf("%s.%s", walletID, typ)
}

type locker struct {
	Locker
	txID transaction.ID
}

func (l *locker) TryLock(ctx context.Context, tokenID *token2.ID) bool {
	err := l.Lock(ctx, tokenID, l.txID)
	if err != nil {
		logger.DebugfContext(ctx, "failed to lock [%v] for [%s]: [%s]", tokenID, l.txID, err)
	}

	return err == nil
}

func (l *locker) UnlockAll(ctx context.Context) error {
	return l.UnlockByTxID(ctx, l.txID)
}

func NewSherdSelector(txID transaction.ID, fetcher TokenFetcher, lockDB Locker, precision uint64, backoff time.Duration, maxRetriesAfterBackoff int, maxTokensPerSelection int, maxLockAttempts int, maxRetryCycles int, selectionTimeout time.Duration, m *Metrics) TokenSelectorUnlocker {
	logger := logger.Named("selector-" + txID)
	locker := &locker{txID: txID, Locker: lockDB}
	if backoff < 0 {
		return NewSelector(logger, fetcher, locker, precision, maxTokensPerSelection, maxLockAttempts, selectionTimeout, m)
	} else {
		return NewStubbornSelector(logger, fetcher, locker, precision, backoff, maxRetriesAfterBackoff, maxTokensPerSelection, maxLockAttempts, maxRetryCycles, selectionTimeout, m)
	}
}
