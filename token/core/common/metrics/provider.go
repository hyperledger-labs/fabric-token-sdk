/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
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
	defer func() { recoverFromDuplicate(recover()) }()

	return p.provider.NewCounter(o).With(p.tmsLabels...)
}

func (p *tmsProvider) NewGauge(o GaugeOpts) Gauge {
	defer func() { recoverFromDuplicate(recover()) }()

	return p.provider.NewGauge(o).With(p.tmsLabels...)
}

func (p *tmsProvider) NewHistogram(o HistogramOpts) Histogram {
	defer func() { recoverFromDuplicate(recover()) }()

	return p.provider.NewHistogram(o).With(p.tmsLabels...)
}

func recoverFromDuplicate(recovered any) {
	if recovered == nil {
		// Registered successfully
		return
	}
	if err, ok := recovered.(error); ok && errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		// Different TMS's try to register the same metric
		logger.Warnf("Recovered from panic: %v\n", err)

		return
	}
	panic(recovered)
}
