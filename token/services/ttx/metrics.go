/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"reflect"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
)

var (
	spKey = reflect.TypeOf((*Metrics)(nil))

	endorsedTransactions = metrics.CounterOpts{
		Namespace:    "ttx",
		Name:         "endorsed_transactions",
		Help:         "The number of endorsed transactions.",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	auditApprovedTransactions = metrics.CounterOpts{
		Namespace:    "ttx",
		Name:         "audit_approved_transactions",
		Help:         "The number of approved transactions by the auditor.",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	acceptedTransactions = metrics.CounterOpts{
		Namespace:    "ttx",
		Name:         "accepted_transactions",
		Help:         "The number of accepted transactions.",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
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

func GetMetrics(sp view2.ServiceProvider) *Metrics {
	s, err := sp.GetService(spKey)
	if err != nil {
		panic(err)
	}
	return s.(*Metrics)
}
