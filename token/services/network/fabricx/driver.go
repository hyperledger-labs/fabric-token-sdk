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
	finalityx "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/finality"
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
	endorsement2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/endorsement"
	finality2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/queue"
	lookup2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/lookup"
	pp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/qe"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
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
	storeServiceManager ttxdb.StoreServiceManager,
	queryServiceProvider queryservice.Provider,
	finalityProvider *finalityx.Provider,
) (driver.Driver, error) {
	vkp := pp2.NewVersionKeeperProvider()
	kt := &keys.Translator{}

	queryExecutorProvider := qe.NewExecutorProvider(qsProvider)

	// In FabricX, we only support 'notification' finality type
	lmCfg := config2.NewListenerManagerConfig(configService)
	if lmCfg.Type() != config3.Notification {
		return nil, errors.Errorf("invalid finality type [%s], expected [%s]", lmCfg.Type(), config3.Notification)
	}

	// Load event queue configuration from token.finality.notification
	qCfg := queue.NewConfig(configService)
	q, err := queue.NewEventQueue(queue.Config{
		Workers:   qCfg.Workers(),
		QueueSize: qCfg.QueueSize(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating event queue")
	}

	flmProvider, err := finality2.NewNotificationServiceBased(queryServiceProvider, finalityProvider, q)
	if err != nil {
		return nil, errors.Wrapf(err, "failed initializing finality provider")
	}

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
		flmProvider:                flmProvider,
		llmProvider: lookup2.NewListenerManagerProvider(
			fnsProvider,
			tracerProvider,
			kt,
			config3.NewListenerManagerConfig(configService),
		),
		EndorsementServiceProvider: endorsement.NewServiceProvider(
			configs,
			viewManager,
			viewManager,
			identityProvider,
			kt,
			vkp,
			tmsProvider,
			endorsement2.NewStorageProvider(storeServiceManager),
			fnsProvider,
		),
		setupListenerProvider: lookup2.NewSetupListenerProvider(
			tmsProvider,
			tokensManager,
			vkp,
		),
		supportedDrivers: []string{fabricx.FabricxDriverName},
	}

	return d, nil
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
