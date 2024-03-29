/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package simple

import (
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
	tracer               Tracer
}

func NewManager(
	locker Locker,
	newQueryEngine NewQueryEngineFunc,
	numRetry int,
	timeout time.Duration,
	requestCertification bool,
	precision uint64,
	tracer Tracer,
) *Manager {
	return &Manager{
		locker:               locker,
		newQueryEngine:       newQueryEngine,
		numRetry:             numRetry,
		timeout:              timeout,
		requestCertification: requestCertification,
		precision:            precision,
		tracer:               tracer,
	}
}

func (m *Manager) NewSelector(id string) (token.Selector, error) {
	return &selector{
		txID:                 id,
		locker:               m.locker,
		queryService:         m.newQueryEngine(),
		precision:            m.precision,
		numRetry:             m.numRetry,
		timeout:              m.timeout,
		requestCertification: m.requestCertification,
		tracer:               m.tracer,
	}, nil
}

func (m *Manager) Unlock(txID string) error {
	m.locker.UnlockByTxID(txID)
	return nil
}
