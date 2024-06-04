/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type (
	CounterOpts = metrics.CounterOpts
	Counter     = metrics.Counter

	GaugeOpts = metrics.GaugeOpts
	Gauge     = metrics.Gauge

	HistogramOpts = metrics.HistogramOpts
	Histogram     = metrics.Histogram

	Provider = metrics.Provider
)

var (
	issues = metrics.CounterOpts{
		Namespace:    "token_sdk",
		Name:         "issue_operations",
		Help:         "The number of issue operations",
		LabelNames:   []string{"network", "channel", "namespace", "token_type"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}.%{token_type}",
	}
	transfers = metrics.CounterOpts{
		Namespace:    "token_sdk",
		Name:         "transfer_operations",
		Help:         "The number of transfer operations",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
)

type Metrics struct {
	provider Provider
	labels   []string

	Issues    Counter
	Transfers Counter
}

func New(provider Provider, tmsID token.TMSID) *Metrics {
	m := &Metrics{
		provider: provider,
		labels: []string{
			"network", tmsID.Network,
			"channel", tmsID.Channel,
			"namespace", tmsID.Namespace,
		},
	}
	m.Issues = m.NewCounter(issues)
	m.Transfers = m.NewCounter(transfers)
	return m
}

func (m *Metrics) AddIssue(tokenType string) {
	m.Issues.With("token_type", tokenType).Add(1)
}

func (m *Metrics) AddTransfer() {
	m.Transfers.With().Add(1)
}

func (m *Metrics) NewCounter(opts CounterOpts) Counter {
	return &counter{rootLabels: m.labels, Counter: m.provider.NewCounter(opts)}
}

func (m *Metrics) NewGauge(opts GaugeOpts) Gauge {
	return m.provider.NewGauge(opts)
}

func (m *Metrics) NewHistogram(opts HistogramOpts) Histogram {
	return m.provider.NewHistogram(opts)
}

type counter struct {
	rootLabels []string
	Counter
}

func (c *counter) With(labels ...string) Counter {
	l := make([]string, len(c.rootLabels)+len(labels))
	l = append(l, c.rootLabels...)
	l = append(l, labels...)
	return c.Counter.With(l...)
}
