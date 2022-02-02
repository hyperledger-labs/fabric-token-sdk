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
	certClient           CertClient
	numRetry             int
	timeout              time.Duration
	requestCertification bool
	metricsAgent         MetricsAgent
}

func newManager(
	locker Locker,
	newQueryEngine NewQueryEngineFunc,
	certClient CertClient,
	numRetry int,
	timeout time.Duration,
	requestCertification bool,
	metricsAgent MetricsAgent,
) *manager {
	return &manager{
		locker:               locker,
		newQueryEngine:       newQueryEngine,
		certClient:           certClient,
		numRetry:             numRetry,
		timeout:              timeout,
		requestCertification: requestCertification,
		metricsAgent:         metricsAgent,
	}
}

func (m *manager) NewSelector(id string) (token.Selector, error) {
	return newSelector(
		id,
		m.locker,
		m.newQueryEngine(),
		m.certClient,
		m.numRetry,
		m.timeout,
		m.requestCertification,
		m.metricsAgent,
	), nil
}

func (m *manager) Unlock(txID string) error {
	m.locker.UnlockByTxID(txID)
	return nil
}
