/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/metrics"
)

const (
	fetcherTypeLabel tracing.LabelName = "fetcher_type"
	lazy             string            = "lazy"
	eager            string            = "eager"
)

type Metrics struct {
	UnspentTokensInvocations metrics.Counter
}

func newMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		UnspentTokensInvocations: p.NewCounter(metrics.CounterOpts{
			Namespace:  "sherdlock",
			Name:       "unspent_tokens_invocations",
			Help:       "The number of invocations",
			LabelNames: []string{fetcherTypeLabel},
		}),
	}
}
