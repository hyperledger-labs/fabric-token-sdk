/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabricx

import (
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/vault/queryservice"
	fabricx "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	config3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/endorsement"
	finality2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality"
	lookup2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/lookup"
	pp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/qe"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"go.opentelemetry.io/otel/trace"
)

func NewDriver(
	fnsProvider *fabric2.NetworkServiceProvider,
	tokensManager *tokens.ServiceManager,
	configs *config.Service,
	viewManager *view.Manager,
	filterProvider *common.AcceptTxInDBFilterProvider,
	tmsProvider *token.ManagementServiceProvider,
	tracerProvider trace.TracerProvider,
	identityProvider view.IdentityProvider,
	ppFetcher *pp2.PublicParametersService,
	configService driver2.ConfigService,
	qsProvider queryservice.Provider,
) driver.Driver {
	vkp := pp2.NewVersionKeeperProvider()
	kt := &keys.Translator{}

	queryExecutorProvider := qe.NewExecutorProvider(qsProvider)

	listenerManagerConfig := config3.NewListenerManagerConfig(configService)

	d := &Driver{
		fnsProvider:                fnsProvider,
		tokensManager:              tokensManager,
		configService:              configs,
		viewManager:                viewManager,
		filterProvider:             filterProvider,
		tmsProvider:                tmsProvider,
		tracerProvider:             tracerProvider,
		identityProvider:           identityProvider,
		defaultPublicParamsFetcher: ppFetcher,
		queryExecutorProvider:      queryExecutorProvider,
		keyTranslator:              kt,
		flmProvider:                finality2.NewFLMProvider(fnsProvider, tracerProvider, listenerManagerConfig),
		llmProvider: lookup2.NewListenerManagerProvider(
			fnsProvider,
			tracerProvider,
			kt,
			config3.NewListenerManagerConfig(configService),
		),
		EndorsementServiceProvider: endorsement.NewServiceProvider(
			fnsProvider,
			configs,
			viewManager,
			viewManager,
			identityProvider,
			kt,
			vkp,
		),
		setupListenerProvider: lookup2.NewSetupListenerProvider(
			tmsProvider,
			tokensManager,
			vkp,
		),
		supportedDrivers: []string{fabricx.FabricxDriverName},
	}

	return d
}

type Driver struct {
	fnsProvider                *fabric2.NetworkServiceProvider
	tokensManager              *tokens.ServiceManager
	configService              *config.Service
	viewManager                *view.Manager
	filterProvider             *common.AcceptTxInDBFilterProvider
	tmsProvider                *token.ManagementServiceProvider
	tracerProvider             trace.TracerProvider
	identityProvider           view.IdentityProvider
	defaultPublicParamsFetcher fabric.NetworkPublicParamsFetcher
	supportedDrivers           []string
	keyTranslator              translator.KeyTranslator
	flmProvider                finality.ListenerManagerProvider
	llmProvider                lookup.ListenerManagerProvider
	EndorsementServiceProvider fabric.EndorsementServiceProvider
	setupListenerProvider      fabric.SetupListenerProvider
	queryExecutorProvider      *qe.ExecutorProvider
}

func (d *Driver) New(network, channel string) (driver.Network, error) {
	fns, err := d.fnsProvider.FabricNetworkService(network)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric network [%s] not found", network)
	}
	if !slices.Contains(d.supportedDrivers, fns.ConfigService().DriverName()) {
		return nil, errors.Errorf("only drivers [%s] supported. [%s] provided", d.supportedDrivers, fns.ConfigService().DriverName())
	}
	ch, err := fns.Channel(channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric channel [%s:%s] not found", network, channel)
	}

	tokenQueryExecutor, err := d.queryExecutorProvider.GetExecutor(fns.Name(), ch.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token query executor for [%s:%s]", fns.Name(), ch.Name())
	}
	spentTokenQueryExecutor, err := d.queryExecutorProvider.GetSpentExecutor(fns.Name(), ch.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spent token query executor")
	}
	queryStateExecutor, err := d.queryExecutorProvider.GetStateExecutor(fns.Name(), ch.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get state query executor for [%s:%s]", fns.Name(), ch.Name())
	}
	flm, err := d.flmProvider.NewManager(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create a new flm")
	}
	llm, err := d.llmProvider.NewManager(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create a new llm")
	}

	logger.Debugf("fabricx network [%s:%s] with driver [%s] ready to be created...", network, channel, fns.ConfigService().DriverName())

	return NewNetwork(
		fns,
		ch,
		d.configService,
		d.filterProvider,
		d.tokensManager,
		d.viewManager,
		d.tmsProvider,
		d.EndorsementServiceProvider,
		tokenQueryExecutor,
		d.tracerProvider,
		d.defaultPublicParamsFetcher,
		spentTokenQueryExecutor,
		queryStateExecutor,
		d.keyTranslator,
		flm,
		llm,
		d.setupListenerProvider,
	), nil
}
