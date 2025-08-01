/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"context"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var logger = logging.MustGetLogger()

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
	defer func() { recoverFromDuplicate(context.Background(), recover()) }()
	return p.provider.NewCounter(o).With(p.tmsLabels...)
}

func (p *tmsProvider) NewGauge(o GaugeOpts) Gauge {
	defer func() { recoverFromDuplicate(context.Background(), recover()) }()
	return p.provider.NewGauge(o).With(p.tmsLabels...)
}

func (p *tmsProvider) NewHistogram(o HistogramOpts) Histogram {
	defer func() { recoverFromDuplicate(context.Background(), recover()) }()
	return p.provider.NewHistogram(o).With(p.tmsLabels...)
}

func recoverFromDuplicate(ctx context.Context, recovered any) {
	if recovered == nil {
		// Registered successfully
		return
	}
	if err, ok := recovered.(error); ok && errors.As(err, &prometheus.AlreadyRegisteredError{}) {
		// Different TMS's try to register the same metric
		logger.WarnfContext(ctx, "Recovered from panic: %v\n", err)
		return
	}
	panic(recovered)
}

func AllLabelNames(extraLabels ...MetricLabel) []MetricLabel {
	return append([]string{NetworkLabel, ChannelLabel, NamespaceLabel}, extraLabels...)
}
func StatsdFormat(extraLabels ...MetricLabel) string {
	return "%{#fqname}.%{" + strings.Join(AllLabelNames(extraLabels...), "}.%{") + "}"
}
