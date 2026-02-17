/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

var (
	spKey = reflect.TypeOf((*Metrics)(nil))

	endorsedTransactions = metrics.CounterOpts{
		Name:       "endorsed_transactions",
		Help:       "The number of endorsed transactions.",
		LabelNames: []string{"network", "channel", "namespace"},
	}
	auditApprovedTransactions = metrics.CounterOpts{
		Name:       "audit_approved_transactions",
		Help:       "The number of approved transactions by the auditor.",
		LabelNames: []string{"network", "channel", "namespace"},
	}
	acceptedTransactions = metrics.CounterOpts{
		Name:       "accepted_transactions",
		Help:       "The number of accepted transactions.",
		LabelNames: []string{"network", "channel", "namespace"},
	}
)

type Metrics struct {
	EndorsedTransactions      metrics.Counter
	AuditApprovedTransactions metrics.Counter
	AcceptedTransactions      metrics.Counter
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		EndorsedTransactions:      p.NewCounter(endorsedTransactions),
		AuditApprovedTransactions: p.NewCounter(auditApprovedTransactions),
		AcceptedTransactions:      p.NewCounter(acceptedTransactions),
	}
}

func GetMetrics(sp token.ServiceProvider) *Metrics {
	s, err := sp.GetService(spKey)
	if err != nil {
		panic(err)
	}

	return s.(*Metrics)
}
