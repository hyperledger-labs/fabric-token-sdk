/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type committerBasedFLMProvider struct {
	onsp           *orion.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	viewManager    *view2.Manager
}

func NewCommitterBasedFLMProvider(onsp *orion.NetworkServiceProvider, tracerProvider trace.TracerProvider, viewManager *view2.Manager) *committerBasedFLMProvider {
	return &committerBasedFLMProvider{
		onsp:           onsp,
		tracerProvider: tracerProvider,
		viewManager:    viewManager,
	}
}

func (p *committerBasedFLMProvider) NewManager(network string, dbManager *DBManager) (FinalityListenerManager, error) {
	net, err := p.onsp.NetworkService(network)
	if err != nil {
		return nil, err
	}
	return &committerBasedFLM{
		net: net,
		tracer: p.tracerProvider.Tracer("finality_listener_manager", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace: network,
		})),
		subscribers: events.NewSubscribers(),
		viewManager: p.viewManager,
		dbManager:   dbManager,
	}, nil
}

type committerBasedFLM struct {
	net         *orion.NetworkService
	tracer      trace.Tracer
	subscribers *events.Subscribers
	viewManager *view2.Manager
	dbManager   *DBManager
}

func (m *committerBasedFLM) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	wrapper := &FinalityListener{
		root:        listener,
		network:     m.net.Name(),
		namespace:   namespace,
		retryRunner: db.NewRetryRunner(3, 100*time.Millisecond, true),
		viewManager: m.viewManager,
		dbManager:   m.dbManager,
		tracer:      m.tracer,
	}
	m.subscribers.Set(txID, listener, wrapper)
	return m.net.Committer().AddFinalityListener(txID, wrapper)
}

func (m *committerBasedFLM) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	wrapper, ok := m.subscribers.Get(txID, listener)
	if !ok {
		return errors.Errorf("listener was not registered")
	}
	return m.net.Committer().RemoveFinalityListener(txID, wrapper.(*FinalityListener))
}
