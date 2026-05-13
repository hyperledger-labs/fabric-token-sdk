/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package simple

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type NewQueryEngineFunc func() QueryService

type Manager struct {
	locker               Locker
	newQueryEngine       NewQueryEngineFunc
	numRetry             int
	timeout              time.Duration
	requestCertification bool
	precision            uint64

	// Resource limits
	maxTokensPerSelection  int
	maxLockAttempts        int
	maxRetryCycles         int
	selectionTimeout       time.Duration
}

func NewManager(
	locker Locker,
	newQueryEngine NewQueryEngineFunc,
	numRetry int,
	timeout time.Duration,
	requestCertification bool,
	precision uint64,
	maxTokensPerSelection int,
	maxLockAttempts int,
	maxRetryCycles int,
	selectionTimeout time.Duration,
) *Manager {
	return &Manager{
		locker:                 locker,
		newQueryEngine:         newQueryEngine,
		numRetry:               numRetry,
		timeout:                timeout,
		requestCertification:   requestCertification,
		precision:              precision,
		maxTokensPerSelection:  maxTokensPerSelection,
		maxLockAttempts:        maxLockAttempts,
		maxRetryCycles:         maxRetryCycles,
		selectionTimeout:       selectionTimeout,
	}
}

func (m *Manager) NewSelector(id string) (token.Selector, error) {
	return &selector{
		txID:                  id,
		locker:                m.locker,
		queryService:          m.newQueryEngine(),
		precision:             m.precision,
		numRetry:              m.numRetry,
		timeout:               m.timeout,
		requestCertification:  m.requestCertification,
		maxTokensPerSelection: m.maxTokensPerSelection,
		maxLockAttempts:       m.maxLockAttempts,
		maxRetryCycles:        m.maxRetryCycles,
		selectionTimeout:      m.selectionTimeout,
	}, nil
}

func (m *Manager) Unlock(ctx context.Context, txID string) error {
	m.locker.UnlockByTxID(ctx, txID)

	return nil
}

func (m *Manager) Close(txID string) error { return nil }
