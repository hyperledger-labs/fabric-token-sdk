/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type committerBasedFLMProvider struct {
	fnsp           *fabric.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	keyTranslator  translator.KeyTranslator
}

func NewCommitterBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator) *committerBasedFLMProvider {
	return &committerBasedFLMProvider{
		fnsp:           fnsp,
		tracerProvider: tracerProvider,
		keyTranslator:  keyTranslator,
	}
}

func (p *committerBasedFLMProvider) NewManager(network, channel string) (FinalityListenerManager, error) {
	net, err := p.fnsp.FabricNetworkService(network)
	if err != nil {
		return nil, err
	}
	ch, err := net.Channel(channel)
	if err != nil {
		return nil, err
	}
	return &committerBasedFLM{
		network:     network,
		channel:     ch,
		subscribers: events.NewSubscribers(),
		tracer: p.tracerProvider.Tracer("finality_listener_manager", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace: network,
		})),
		keyTranslator: p.keyTranslator,
	}, nil
}

type committerBasedFLM struct {
	network       string
	channel       *fabric.Channel
	tracer        trace.Tracer
	subscribers   *events.Subscribers
	keyTranslator translator.KeyTranslator
}

func (m *committerBasedFLM) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	wrapper := &FinalityListener{
		root:          listener,
		flm:           m,
		network:       m.network,
		ch:            m.channel,
		namespace:     namespace,
		tracer:        m.tracer,
		keyTranslator: m.keyTranslator,
	}
	m.subscribers.Set(txID, listener, wrapper)
	return m.channel.Committer().AddFinalityListener(txID, wrapper)
}

func (m *committerBasedFLM) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	wrapper, ok := m.subscribers.Get(txID, listener)
	if !ok {
		return errors.Errorf("listener was not registered")
	}
	return m.channel.Committer().RemoveFinalityListener(txID, wrapper.(*FinalityListener))
}
