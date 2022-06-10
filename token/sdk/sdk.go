/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/interactive"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/badger"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/memory"
)

var logger = flogging.MustGetLogger("token-sdk")

type Registry interface {
	GetService(v interface{}) (interface{}, error)

	RegisterService(service interface{}) error
}

type SDK struct {
	registry Registry
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

	logger.Infof("Set TMS Provider")
	pm := NewProcessorManager(p.registry)
	tmsProvider := core.NewTMSProvider(
		p.registry,
		pm.New,
	)
	assert.NoError(p.registry.RegisterService(tmsProvider))

	// Register the token management service provider
	assert.NoError(p.registry.RegisterService(token.NewManagementServiceProvider(
		p.registry,
		tmsProvider,
		network2.NewNormalizer(config.NewTokenSDK(configProvider), p.registry),
		vault.NewVaultProvider(p.registry),
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

	// DB
	driverName := view2.GetConfigService(p.registry).GetString("token.auditor.ttxdb.persistence.type")
	if len(driverName) == 0 {
		driverName = "memory"
	}
	assert.NoError(p.registry.RegisterService(ttxdb.NewManager(p.registry, driverName)))

	logger.Infof("Install View Handlers")
	query.InstallQueryViewFactories(p.registry)

	enabled, err := orion2.IsCustodian(view2.GetConfigService(p.registry))
	assert.NoError(err, "failed to get custodian status")
	logger.Infof("Orion Custodian enabled: %t", enabled)
	if enabled {
		assert.NoError(orion2.InstallViews(p.registry), "failed to install custodian views")
	}

	return nil
}

func (p *SDK) Start(ctx context.Context) error {
	return nil
}
