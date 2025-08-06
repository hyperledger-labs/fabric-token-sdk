/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"go.opentelemetry.io/otel/trace"
)

type CommitterListenerManagerConfig struct {
	MaxRetries        int
	RetryWaitDuration time.Duration
}

type committerBasedFLMProvider struct {
	fnsp           *fabric.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	keyTranslator  translator.KeyTranslator
	config         CommitterListenerManagerConfig
}

func NewCommitterBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, config CommitterListenerManagerConfig) *committerBasedFLMProvider {
	return &committerBasedFLMProvider{
		fnsp:           fnsp,
		tracerProvider: tracerProvider,
		keyTranslator:  keyTranslator,
		config:         config,
	}
}

func (p *committerBasedFLMProvider) NewManager(network, channel string) (ListenerManager, error) {
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
		keyTranslator:     p.keyTranslator,
		maxRetries:        p.config.MaxRetries,
		retryWaitDuration: p.config.RetryWaitDuration,
	}, nil
}

type committerBasedFLM struct {
	network           string
	channel           *fabric.Channel
	tracer            trace.Tracer
	subscribers       *events.Subscribers
	keyTranslator     translator.KeyTranslator
	maxRetries        int
	retryWaitDuration time.Duration
}

func (m *committerBasedFLM) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	wrapper := &FinalityListener{
		root:              listener,
		flm:               m,
		network:           m.network,
		ch:                m.channel,
		namespace:         namespace,
		tracer:            m.tracer,
		keyTranslator:     m.keyTranslator,
		maxRetries:        m.maxRetries,
		retryWaitDuration: m.retryWaitDuration,
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

type FinalityListener struct {
	flm               driver.FinalityListenerManager
	root              driver.FinalityListener
	network           string
	ch                *fabric.Channel
	namespace         string
	tracer            trace.Tracer
	keyTranslator     translator.KeyTranslator
	maxRetries        int
	retryWaitDuration time.Duration
}

func (t *FinalityListener) OnStatus(ctx context.Context, txID string, status int, message string) {
	newCtx, span := t.tracer.Start(ctx, "on_status")
	defer span.End()
	defer func() {
		if e := recover(); e != nil {
			span.RecordError(fmt.Errorf("recovered from panic: %v", e))
			logger.DebugfContext(ctx, "failed finality update for tx [%s]: [%s]", txID, e)
			if err := t.flm.AddFinalityListener(txID, t.namespace, t.root); err != nil {
				panic(err)
			}
			logger.DebugfContext(ctx, "added finality listener for tx [%s]...done", txID)
		}
	}()

	key, err := t.keyTranslator.CreateTokenRequestKey(txID)
	if err != nil {
		panic(fmt.Sprintf("can't create for token request [%s]", txID))
	}

	v := t.ch.Vault()
	qe, err := v.NewQueryExecutor(ctx)
	if err != nil {
		panic(fmt.Sprintf("can't get query executor [%s]", txID))
	}

	// Fetch the token request hash. Retry in case some other replica committed it shortly before
	logger.DebugfContext(ctx, "fetch token request hash")
	var tokenRequestHash *driver2.VaultRead
	var retries int
	for tokenRequestHash, err = qe.GetState(ctx, t.namespace, key); err == nil && (tokenRequestHash == nil || len(tokenRequestHash.Raw) == 0) && retries < t.maxRetries; tokenRequestHash, err = qe.GetState(ctx, t.namespace, key) {
		logger.DebugfContext(ctx, "did not find token request [%s]. retrying...", txID)
		retries++
		time.Sleep(t.retryWaitDuration)
	}
	if err := qe.Done(); err != nil {
		logger.Warnf("failed to close query executor for tx [%s]: [%s]", txID, err)
	}
	if err != nil {
		panic(fmt.Sprintf("can't get state [%s][%s]", txID, key))
	}
	logger.DebugfContext(ctx, "fetch token request hash done, emit event")
	t.root.OnStatus(newCtx, txID, status, message, tokenRequestHash.Raw)
}
