/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
)

var (
	zkIssueDurationOpts = metrics.HistogramOpts{
		Namespace:    "token_sdk.zkatdlog.nogh",
		Name:         "issue_duration",
		Help:         "Duration of zk issue token",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
	zkTransferDurationOpts = metrics.HistogramOpts{
		Namespace:    "token_sdk.zkatdlog.nogh",
		Name:         "transfer_duration",
		Help:         "Duration of zk transfer token",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
)

type Metrics struct {
	zkIssueDuration    metrics.Histogram
	zkTransferDuration metrics.Histogram
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		zkIssueDuration:    p.NewHistogram(zkIssueDurationOpts),
		zkTransferDuration: p.NewHistogram(zkTransferDurationOpts),
	}
}
