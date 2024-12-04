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
	vault2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type FinalityListenerManagerProvider interface {
	NewManager(network string, dbManager *DBManager) (FinalityListenerManager, error)
}

type FinalityListenerManager driver.FinalityListenerManager

func NewOrionDriver(
	onsProvider *orion.NetworkServiceProvider,
	viewRegistry driver2.Registry,
	viewManager *view.Manager,
	vaultProvider *vault2.Provider,
	configProvider *view.ConfigService,
	configService *config.Service,
	identityProvider view2.IdentityProvider,
	filterProvider *common.AcceptTxInDBFilterProvider,
	tmsProvider *token.ManagementServiceProvider,
	tracerProvider trace.TracerProvider,
) driver.Driver {
	keyTranslator := &translator.HashedKeyTranslator{KT: &keys.Translator{}}
	return NewDriver(
		onsProvider,
		viewRegistry,
		viewManager,
		vaultProvider,
		configProvider,
		configService,
		identityProvider,
		filterProvider,
		tmsProvider,
		NewTokenExecutorProvider(),
		NewSpentTokenExecutorProvider(keyTranslator),
		tracerProvider,
		keyTranslator,
		NewCommitterBasedFLMProvider(onsProvider, tracerProvider, viewManager),
	)
}

type Driver struct {
	onsProvider                     *orion.NetworkServiceProvider
	viewRegistry                    driver2.Registry
	viewManager                     *view.Manager
	vaultProvider                   driver.TokenVaultProvider
	configProvider                  configProvider
	configService                   *config.Service
	identityProvider                view2.IdentityProvider
	filterProvider                  *common.AcceptTxInDBFilterProvider
	tmsProvider                     *token.ManagementServiceProvider
	tokenQueryExecutorProvider      driver.TokenQueryExecutorProvider
	spentTokenQueryExecutorProvider driver.SpentTokenQueryExecutorProvider
	tracerProvider                  trace.TracerProvider
	keyTranslator                   translator.KeyTranslator
	flmProvider                     FinalityListenerManagerProvider
}

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
	tokenQueryExecutorProvider driver.TokenQueryExecutorProvider,
	spentTokenQueryExecutorProvider driver.SpentTokenQueryExecutorProvider,
	tracerProvider trace.TracerProvider,
	keyTranslator translator.KeyTranslator,
	flmProvider FinalityListenerManagerProvider,
) *Driver {
	return &Driver{
		onsProvider:                     onsProvider,
		viewRegistry:                    viewRegistry,
		viewManager:                     viewManager,
		vaultProvider:                   vaultProvider,
		configProvider:                  configProvider,
		configService:                   configService,
		identityProvider:                identityProvider,
		filterProvider:                  filterProvider,
		tmsProvider:                     tmsProvider,
		tokenQueryExecutorProvider:      tokenQueryExecutorProvider,
		spentTokenQueryExecutorProvider: spentTokenQueryExecutorProvider,
		tracerProvider:                  tracerProvider,
		keyTranslator:                   keyTranslator,
		flmProvider:                     flmProvider,
	}
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

	tokenQueryExecutor, err := d.tokenQueryExecutorProvider.GetExecutor(n.Name(), "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token query executor")
	}
	spentTokenQueryExecutor, err := d.spentTokenQueryExecutorProvider.GetSpentExecutor(n.Name(), "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get spent token query executor")
	}
	flm, err := d.flmProvider.NewManager(n.Name(), dbManager)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get flm")
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
		flm,
		tokenQueryExecutor,
		spentTokenQueryExecutor,
		d.tracerProvider,
		d.keyTranslator,
	), nil
}
