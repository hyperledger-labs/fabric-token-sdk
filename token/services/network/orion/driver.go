/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	vault2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type DefaultPublicParamsFetcher driver3.DefaultPublicParamsFetcher

type TokenQueryExecutorProvider driver3.TokenQueryExecutorProvider

type SpentTokenQueryExecutorProvider driver3.SpentTokenQueryExecutorProvider

func NewDriver(
	onsProvider *orion.NetworkServiceProvider,
	viewRegistry driver2.Registry,
	viewManager *view.Manager,
	vaultProvider *vault2.Provider,
	configProvider *view.ConfigService,
	configService *config.Service,
	identityProvider view2.IdentityProvider,
	filterProvider *common.AcceptTxInDBFilterProvider,
	tmsProvider *token.ManagementServiceProvider,
	defaultPublicParamsFetcher DefaultPublicParamsFetcher,
	tokenQueryExecutorProvider TokenQueryExecutorProvider,
	spentTokenQueryExecutorProvider SpentTokenQueryExecutorProvider,
	tracerProvider trace.TracerProvider,
) driver.NamedDriver {
	return driver.NamedDriver{
		Name: "orion",
		Driver: &Driver{
			onsProvider:                     onsProvider,
			viewRegistry:                    viewRegistry,
			viewManager:                     viewManager,
			vaultProvider:                   vaultProvider,
			configProvider:                  configProvider,
			configService:                   configService,
			identityProvider:                identityProvider,
			filterProvider:                  filterProvider,
			tmsProvider:                     tmsProvider,
			defaultPublicParamsFetcher:      defaultPublicParamsFetcher,
			tokenQueryExecutorProvider:      tokenQueryExecutorProvider,
			spentTokenQueryExecutorProvider: spentTokenQueryExecutorProvider,
			tracerProvider:                  tracerProvider,
		},
	}
}

type Driver struct {
	onsProvider                     *orion.NetworkServiceProvider
	viewRegistry                    driver2.Registry
	viewManager                     *view.Manager
	vaultProvider                   vault.Provider
	configProvider                  configProvider
	configService                   *config.Service
	identityProvider                view2.IdentityProvider
	filterProvider                  *common.AcceptTxInDBFilterProvider
	tmsProvider                     *token.ManagementServiceProvider
	defaultPublicParamsFetcher      driver3.DefaultPublicParamsFetcher
	tokenQueryExecutorProvider      TokenQueryExecutorProvider
	spentTokenQueryExecutorProvider SpentTokenQueryExecutorProvider
	tracerProvider                  trace.TracerProvider
}

func (d *Driver) New(network, _ string) (driver.Network, error) {
	n, err := d.onsProvider.NetworkService(network)
	if err != nil {
		return nil, errors.WithMessagef(err, "network [%s] not found", network)
	}

	enabled, err := IsCustodian(d.configProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed checking if custodian is enabled")
	}
	logger.Infof("Orion Custodian enabled: %t", enabled)
	dbManager := NewDBManager(d.onsProvider, d.configProvider, enabled)
	statusCache := secondcache.NewTyped[*TxStatusResponse](1000)
	if enabled {
		if err := InstallViews(d.viewRegistry, dbManager, statusCache); err != nil {
			return nil, errors.WithMessagef(err, "failed installing views")
		}
	}

	tokenQueryExecutor, err := d.tokenQueryExecutorProvider.GetExecutor(network, "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token query executor")
	}
	spentTokenQueryExecutor, err := d.spentTokenQueryExecutorProvider.GetSpentExecutor(network, "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spent token query executor")
	}
	return NewNetwork(
		d.viewManager,
		d.tmsProvider,
		d.identityProvider,
		n,
		d.vaultProvider.Vault,
		d.configService,
		d.filterProvider,
		dbManager,
		d.defaultPublicParamsFetcher,
		tokenQueryExecutor,
		spentTokenQueryExecutor,
		d.tracerProvider,
	), nil
}
