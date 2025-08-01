/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
)

var (
	certifiedTokens = metrics.CounterOpts{
		Namespace:    "certification_interactive",
		Name:         "certified_tokens",
		Help:         "The number of tokens certified.",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
)

type Metrics struct {
	CertifiedTokens metrics.Counter
}

func NewMetrics(ctx context.Context, p metrics.Provider) *Metrics {
	return &Metrics{
		CertifiedTokens: p.NewCounter(certifiedTokens),
	}
}
