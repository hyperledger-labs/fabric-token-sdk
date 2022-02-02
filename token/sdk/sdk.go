/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/db/badger"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/db/memory"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/interactive"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	fabric4 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/query"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector"
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
	tmsProvider := core.NewTMSProvider(
		p.registry,
		func(network, channel, namespace string) error {
			n := fabric.GetFabricNetworkService(p.registry, network)
			if err := n.ProcessorManager().AddProcessor(
				namespace,
				fabric4.NewTokenRWSetProcessor(
					n,
					namespace,
					p.registry,
					fabric4.NewOwnershipMultiplexer(&fabric4.WalletOwnership{}),
					fabric4.NewIssuedMultiplexer(&fabric4.WalletIssued{}),
				),
			); err != nil {
				return errors.Wrapf(err, "failed adding transaction processors")
			}
			return nil
		},
	)
	assert.NoError(p.registry.RegisterService(tmsProvider))

	assert.NoError(p.registry.RegisterService(token.NewManagementServiceProvider(
		p.registry,
		tmsProvider,
		fabric2.NewNormalizer(p.registry),
		fabric2.NewVaultProvider(p.registry),
		fabric2.NewCertificationClientProvider(p.registry),
		selector.NewProvider(
			p.registry,
			fabric2.NewLockerProvider(
				p.registry,
				2*time.Second,
				(5*time.Minute).Milliseconds(),
			),
			2,
			5*time.Second),
	)))

	// Network provider
	assert.NoError(p.registry.RegisterService(network.NewProvider(p.registry)))

	// AuditDB
	driverName := view2.GetConfigService(p.registry).GetString("token.auditor.auditdb.persistence.type")
	if len(driverName) == 0 {
		driverName = "memory"
	}
	assert.NoError(p.registry.RegisterService(auditdb.NewManager(p.registry, driverName)))

	logger.Infof("Install View Handlers")
	query.InstallQueryViewFactories(p.registry)

	return nil
}

func (p *SDK) Start(ctx context.Context) error {
	return nil
}
