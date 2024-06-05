/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type (
	CounterOpts   = metrics.CounterOpts
	Counter       = metrics.Counter
	GaugeOpts     = metrics.GaugeOpts
	Gauge         = metrics.Gauge
	HistogramOpts = metrics.HistogramOpts
	Histogram     = metrics.Histogram
	Provider      = metrics.Provider
)

var GetProvider = metrics.GetProvider

type Metrics struct {
	provider Provider
	labels   []string

	Issues        Counter
	FailedIssues  Counter
	IssueDuration Histogram

	Transfers        Counter
	FailedTransfers  Counter
	TransferDuration Histogram

	Audits        Counter
	FailedAudits  Counter
	AuditDuration Histogram
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
	m.Issues = m.NewCounter(issuesOpts)
	m.FailedIssues = m.NewCounter(failedIssuesOpts)
	m.IssueDuration = m.NewHistogram(issueDurationOpts)

	m.Transfers = m.NewCounter(transfersOpts)
	m.FailedTransfers = m.NewCounter(failedTransfersOpts)
	m.TransferDuration = m.NewHistogram(transferDurationOpts)

	m.Audits = m.NewCounter(auditsOpts)
	m.FailedAudits = m.NewCounter(failedAuditsOpts)
	m.AuditDuration = m.NewHistogram(auditDurationOpts)
	return m
}

func (m *Metrics) AddIssue(tokenType string, noErr bool) {
	if noErr {
		m.Issues.With("token_type", tokenType).Add(1)
		return
	}
	m.FailedIssues.With("token_type", tokenType).Add(1)
}

func (m *Metrics) ObserveIssueDuration(duration time.Duration) {
	m.IssueDuration.Observe(float64(duration.Milliseconds()))
}

func (m *Metrics) AddTransfer(noErr bool) {
	if noErr {
		m.Transfers.With().Add(1)
		return
	}
	m.FailedTransfers.With().Add(1)
}

func (m *Metrics) ObserveTransferDuration(duration time.Duration) {
	m.TransferDuration.Observe(float64(duration.Milliseconds()))
}

func (m *Metrics) AddAudit(noErr bool) {
	if noErr {
		m.Audits.With().Add(1)
		return
	}
	m.FailedAudits.With().Add(1)
}

func (m *Metrics) ObserveAuditDuration(duration time.Duration) {
	m.AuditDuration.With().Observe(float64(duration.Milliseconds()))
}

func (m *Metrics) NewCounter(opts CounterOpts) Counter {
	return &counter{rootLabels: m.labels, Counter: m.provider.NewCounter(opts)}
}

func (m *Metrics) NewGauge(opts GaugeOpts) Gauge {
	return m.provider.NewGauge(opts)
}

func (m *Metrics) NewHistogram(opts HistogramOpts) Histogram {
	return &histogram{rootLabels: m.labels, Histogram: m.provider.NewHistogram(opts)}
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

type histogram struct {
	rootLabels []string
	Histogram
}

func (c *histogram) With(labels ...string) Histogram {
	l := make([]string, len(c.rootLabels)+len(labels))
	l = append(l, c.rootLabels...)
	l = append(l, labels...)
	return c.Histogram.With(l...)
}
