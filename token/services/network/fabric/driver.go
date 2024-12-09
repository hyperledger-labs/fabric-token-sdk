/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	config2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	vault2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type FinalityListenerManagerProvider interface {
	NewManager(network, channel string) (FinalityListenerManager, error)
}

type FinalityListenerManager = driver.FinalityListenerManager

type Driver struct {
	fnsProvider                     *fabric.NetworkServiceProvider
	vaultProvider                   driver.TokenVaultProvider
	tokensManager                   *tokens.Manager
	configService                   *config.Service
	viewManager                     *view.Manager
	viewRegistry                    driver2.Registry
	filterProvider                  *common.AcceptTxInDBFilterProvider
	tmsProvider                     *token.ManagementServiceProvider
	identityProvider                driver2.IdentityProvider
	tracerProvider                  trace.TracerProvider
	defaultPublicParamsFetcher      driver3.NetworkPublicParamsFetcher
	tokenQueryExecutorProvider      driver.TokenQueryExecutorProvider
	spentTokenQueryExecutorProvider driver.SpentTokenQueryExecutorProvider
	supportedDrivers                []string
	keyTranslator                   translator.KeyTranslator
	flmProvider                     FinalityListenerManagerProvider
	EndorsementServiceProvider      EndorsementServiceProvider
}

func NewGenericDriver(
	fnsProvider *fabric.NetworkServiceProvider,
	vaultProvider *vault2.Provider,
	tokensManager *tokens.Manager,
	configService *config.Service,
	viewManager *view.Manager,
	viewRegistry driver2.Registry,
	filterProvider *common.AcceptTxInDBFilterProvider,
	tmsProvider *token.ManagementServiceProvider,
	tracerProvider trace.TracerProvider,
	identityProvider driver2.IdentityProvider,
) driver.Driver {
	keyTranslator := &keys.Translator{}
	return NewDriver(
		fnsProvider,
		vaultProvider,
		tokensManager,
		configService,
		viewManager,
		viewRegistry,
		filterProvider,
		tmsProvider,
		tracerProvider,
		identityProvider,
		NewChaincodePublicParamsFetcher(viewManager),
		NewTokenExecutorProvider(fnsProvider),
		NewSpentTokenExecutorProvider(fnsProvider, keyTranslator),
		keyTranslator,
		NewDeliveryBasedFLMProvider(fnsProvider, tracerProvider, keyTranslator),
		endorsement.NewServiceProvider(fnsProvider, configService, viewManager, viewRegistry, identityProvider, keyTranslator),
		config2.GenericDriver,
	)
}

func NewDriver(
	fnsProvider *fabric.NetworkServiceProvider,
	vaultProvider *vault2.Provider,
	tokensManager *tokens.Manager,
	configService *config.Service,
	viewManager *view.Manager,
	viewRegistry driver2.Registry,
	filterProvider *common.AcceptTxInDBFilterProvider,
	tmsProvider *token.ManagementServiceProvider,
	tracerProvider trace.TracerProvider,
	identityProvider driver2.IdentityProvider,
	defaultPublicParamsFetcher driver3.NetworkPublicParamsFetcher,
	tokenQueryExecutorProvider driver.TokenQueryExecutorProvider,
	spentTokenQueryExecutorProvider driver.SpentTokenQueryExecutorProvider,
	keyTranslator translator.KeyTranslator,
	flmProvider FinalityListenerManagerProvider,
	endorsementServiceProvider EndorsementServiceProvider,
	supportedDrivers ...string,
) *Driver {
	return &Driver{
		fnsProvider:                     fnsProvider,
		vaultProvider:                   vaultProvider,
		tokensManager:                   tokensManager,
		configService:                   configService,
		viewManager:                     viewManager,
		viewRegistry:                    viewRegistry,
		filterProvider:                  filterProvider,
		tmsProvider:                     tmsProvider,
		identityProvider:                identityProvider,
		tracerProvider:                  tracerProvider,
		defaultPublicParamsFetcher:      defaultPublicParamsFetcher,
		tokenQueryExecutorProvider:      tokenQueryExecutorProvider,
		spentTokenQueryExecutorProvider: spentTokenQueryExecutorProvider,
		supportedDrivers:                supportedDrivers,
		keyTranslator:                   keyTranslator,
		flmProvider:                     flmProvider,
		EndorsementServiceProvider:      endorsementServiceProvider,
	}
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

	tokenQueryExecutor, err := d.tokenQueryExecutorProvider.GetExecutor(fns.Name(), ch.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token query executor for [%s:%s]", fns.Name(), ch.Name())
	}
	spentTokenQueryExecutor, err := d.spentTokenQueryExecutorProvider.GetSpentExecutor(fns.Name(), ch.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spent token query executor")
	}
	flm, err := d.flmProvider.NewManager(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create a new flm")
	}

	return NewNetwork(
		fns,
		ch,
		d.vaultProvider.Vault,
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
		d.keyTranslator,
		flm,
	), nil
}
