/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
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
	*metrics.Metrics

	zkIssueDuration    metrics.Histogram
	zkTransferDuration metrics.Histogram
}

func NewMetrics(provider metrics.Provider, tmsID token.TMSID) *Metrics {
	m := &Metrics{Metrics: metrics.New(provider, tmsID)}
	m.zkIssueDuration = m.NewHistogram(zkIssueDurationOpts)
	m.zkTransferDuration = m.NewHistogram(zkTransferDurationOpts)
	return m
}

func (m *Metrics) ObserveZKIssueDuration(duration time.Duration) {
	m.zkIssueDuration.With().Observe(float64(duration.Milliseconds()))
}

func (m *Metrics) ObserveZKTransferDuration(duration time.Duration) {
	m.zkTransferDuration.With().Observe(float64(duration.Milliseconds()))
}
