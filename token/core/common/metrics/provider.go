/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

const (
	NetworkLabel   MetricLabel = "network"
	ChannelLabel   MetricLabel = "channel"
	NamespaceLabel MetricLabel = "namespace"
)

type tmsProvider struct {
	tmsLabels []string
	provider  Provider
}

func NewTMSProvider(tmsID token.TMSID, provider Provider) *tmsProvider {
	return &tmsProvider{
		tmsLabels: []string{
			NetworkLabel, tmsID.Network,
			ChannelLabel, tmsID.Channel,
			NamespaceLabel, tmsID.Namespace,
		},
		provider: provider,
	}
}

func (p *tmsProvider) NewCounter(o CounterOpts) Counter {
	return p.provider.NewCounter(o).With(p.tmsLabels...)
}

func (p *tmsProvider) NewGauge(o GaugeOpts) Gauge { return p.provider.NewGauge(o).With(p.tmsLabels...) }

func (p *tmsProvider) NewHistogram(o HistogramOpts) Histogram {
	return p.provider.NewHistogram(o).With(p.tmsLabels...)
}

func AllLabelNames(extraLabels ...MetricLabel) []MetricLabel {
	return append([]string{NetworkLabel, ChannelLabel, NamespaceLabel}, extraLabels...)
}
func StatsdFormat(extraLabels ...MetricLabel) string {
	return "%{#fqname}.%{" + strings.Join(AllLabelNames(extraLabels...), "}.%{") + "}"
}
