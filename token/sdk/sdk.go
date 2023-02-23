/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tms2 "github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/interactive"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/badger"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk")

type Registry interface {
	GetService(v interface{}) (interface{}, error)

	RegisterService(service interface{}) error
}

type SDK struct {
	registry       Registry
	auditorManager *auditor.Manager
	ownerManager   *owner.Manager
}

func NewSDK(registry Registry) *SDK {
	return &SDK{registry: registry}
}

func (p *SDK) Install() error {
	configProvider := view2.GetConfigService(p.registry)
	if !configProvider.GetBool("token.enabled") {
		logger.Infof("Token platform not enabled, skipping")
		return nil
	}
	logger.Infof("Token platform enabled, installing...")

	logger.Infof("Set TMS TMSProvider")
	vaultProvider := vault.NewProvider(p.registry)
	tmsProvider := tms2.NewTMSProvider(
		p.registry,
		&vault.PublicParamsProvider{Provider: vaultProvider},
		tms.NewPostInitializer(p.registry).PostInit,
	)
	assert.NoError(p.registry.RegisterService(tmsProvider))

	// Register the token management service provider
	assert.NoError(p.registry.RegisterService(token.NewManagementServiceProvider(
		p.registry,
		tmsProvider,
		network2.NewNormalizer(config.NewTokenSDK(configProvider), p.registry),
		vaultProvider,
		network2.NewCertificationClientProvider(p.registry),
		selector.NewProvider(
			p.registry,
			network2.NewLockerProvider(
				p.registry,
				2*time.Second,
				5*time.Minute,
			),
			2,
			5*time.Second),
	)))

	// Network provider
	assert.NoError(p.registry.RegisterService(network.NewProvider(p.registry)))

	// Token Transaction DB and derivatives
	assert.NoError(p.registry.RegisterService(ttxdb.NewManager(p.registry, "")))
	p.auditorManager = auditor.NewManager(p.registry, kvs.GetService(p.registry))
	assert.NoError(p.registry.RegisterService(p.auditorManager))
	p.ownerManager = owner.NewManager(p.registry, kvs.GetService(p.registry))
	assert.NoError(p.registry.RegisterService(p.ownerManager))

	enabled, err := orion.IsCustodian(view2.GetConfigService(p.registry))
	assert.NoError(err, "failed to get custodian status")
	logger.Infof("Orion Custodian enabled: %t", enabled)
	if enabled {
		assert.NoError(orion.InstallViews(p.registry), "failed to install custodian views")
	}

	// Install metrics
	assert.NoError(
		p.registry.RegisterService(ttx.NewMetrics(metrics.GetProvider(p.registry))),
		"failed to register ttx package's metrics",
	)
	return nil
}

func (p *SDK) Start(ctx context.Context) error {
	configProvider := view2.GetConfigService(p.registry)
	if !configProvider.GetBool("token.enabled") {
		logger.Infof("Token platform not enabled, skipping start")
		return nil
	}
	logger.Infof("Token platform enabled, starting...")

	// load the configured tms
	tmsConfigs, err := config.NewTokenSDK(configProvider).GetTMSs()
	if err != nil {
		return errors.WithMessagef(err, "failed get the TMS configurations")
	}
	for _, tmsConfig := range tmsConfigs {
		tmsID := token.TMSID{
			Network:   tmsConfig.TMS().Network,
			Channel:   tmsConfig.TMS().Channel,
			Namespace: tmsConfig.TMS().Namespace,
		}
		tms := token.GetManagementService(p.registry, token.WithTMSID(tmsID))
		if tms == nil {
			return errors.Errorf("failed to load configured TMS [%s]", tmsID)
		}
	}

	// restore owner and auditor dbs, if any
	if err := p.ownerManager.Restore(); err != nil {
		return errors.WithMessagef(err, "failed to restore onwer dbs")
	}
	if err := p.auditorManager.Restore(); err != nil {
		return errors.WithMessagef(err, "failed to restore auditor dbs")
	}

	logger.Infof("Token platform enabled, starting...done")
	return nil
}
