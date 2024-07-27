/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	metrics2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/metrics"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var logger = logging.MustGetLogger("token-sdk.metrics")

const (
	NetworkLabel   metrics2.MetricLabel = "network"
	ChannelLabel   metrics2.MetricLabel = "channel"
	NamespaceLabel metrics2.MetricLabel = "namespace"
)

type tmsProvider struct {
	tmsLabels []string
	provider  metrics2.Provider
}

func NewTMSProvider(tmsID token.TMSID, provider metrics2.Provider) *tmsProvider {
	return &tmsProvider{
		tmsLabels: []string{
			NetworkLabel, tmsID.Network,
			ChannelLabel, tmsID.Channel,
			NamespaceLabel, tmsID.Namespace,
		},
		provider: provider,
	}
}

func (p *tmsProvider) NewCounter(o metrics2.CounterOpts) metrics2.Counter {
	defer func() { recoverFromDuplicate(recover()) }()
	return p.provider.NewCounter(o).With(p.tmsLabels...)
}

func (p *tmsProvider) NewGauge(o metrics2.GaugeOpts) metrics2.Gauge {
	defer func() { recoverFromDuplicate(recover()) }()
	return p.provider.NewGauge(o).With(p.tmsLabels...)
}

func (p *tmsProvider) NewHistogram(o metrics2.HistogramOpts) metrics2.Histogram {
	defer func() { recoverFromDuplicate(recover()) }()
	return p.provider.NewHistogram(o).With(p.tmsLabels...)
}

func recoverFromDuplicate(recovered any) {
	if recovered == nil {
		// Registered successfully
		return
	}
	if err, ok := recovered.(error); ok && errors.As(err, &prometheus.AlreadyRegisteredError{}) {
		// Different TMS's try to register the same metric
		logger.Warnf("Recovered from panic: %v\n", err)
		return
	}
	panic(recovered)

}

func AllLabelNames(extraLabels ...metrics2.MetricLabel) []metrics2.MetricLabel {
	return append([]string{NetworkLabel, ChannelLabel, NamespaceLabel}, extraLabels...)
}
func StatsdFormat(extraLabels ...metrics2.MetricLabel) string {
	return "%{#fqname}.%{" + strings.Join(AllLabelNames(extraLabels...), "}.%{") + "}"
}
