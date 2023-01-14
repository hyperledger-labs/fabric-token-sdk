/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package selector

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type NewQueryEngineFunc func() QueryService

type manager struct {
	locker               Locker
	newQueryEngine       NewQueryEngineFunc
	numRetry             int
	timeout              time.Duration
	requestCertification bool
	precision            uint64
	metricsAgent         Tracer
}

func NewManager(
	locker Locker,
	newQueryEngine NewQueryEngineFunc,
	numRetry int,
	timeout time.Duration,
	requestCertification bool,
	precision uint64,
	metricsAgent Tracer,
) *manager {
	return &manager{
		locker:               locker,
		newQueryEngine:       newQueryEngine,
		numRetry:             numRetry,
		timeout:              timeout,
		requestCertification: requestCertification,
		precision:            precision,
		metricsAgent:         metricsAgent,
	}
}

func (m *manager) NewSelector(id string) (token.Selector, error) {
	return &selector{
		txID:                 id,
		locker:               m.locker,
		queryService:         m.newQueryEngine(),
		precision:            m.precision,
		numRetry:             m.numRetry,
		timeout:              m.timeout,
		requestCertification: m.requestCertification,
		metricsAgent:         m.metricsAgent,
	}, nil
}

func (m *manager) Unlock(txID string) error {
	m.locker.UnlockByTxID(txID)
	return nil
}
