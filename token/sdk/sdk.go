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
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	dbconfig "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/mailman"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk")

type Registry interface {
	GetService(v interface{}) (interface{}, error)

	RegisterService(service interface{}) error
}

type SDK struct {
	registry        Registry
	postInitializer *tms.PostInitializer
	networkProvider *network.Provider
	tmsProvider     *token.ManagementServiceProvider
	configService   *config.Service
}

func NewSDK(registry Registry) *SDK {
	return &SDK{registry: registry}
}

func (p *SDK) Install() error {
	cs := view2.GetConfigService(p.registry)
	assert.NotNil(cs, "config service missing")
	p.configService = config.NewService(cs)
	assert.NoError(p.registry.RegisterService(p.configService))
	if !p.configService.Enabled() {
		logger.Infof("Token platform not enabled, skipping")
		return nil
	}
	logger.Infof("Token platform enabled, installing...")

	logger.Infof("Set configuration TMSProvider")

	// Network provider
	networkProvider := network.NewProvider(p.registry)
	p.networkProvider = networkProvider
	assert.NoError(p.registry.RegisterService(networkProvider))

	tmsProvider := core.NewTMSProvider(
		p.registry,
		flogging.MustGetLogger("token-sdk.core"),
		p.configService,
		&vault.PublicParamsProvider{Provider: networkProvider},
	)
	assert.NoError(p.registry.RegisterService(tmsProvider))

	// DB Managers
	ttxdbManager := ttxdb.NewManager(cs, dbconfig.NewConfig(p.configService, "ttxdb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(ttxdbManager))
	tokenDBManager := tokendb.NewManager(cs, dbconfig.NewConfig(p.configService, "tokendb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(tokenDBManager))
	auditDBManager := auditdb.NewManager(cs, dbconfig.NewConfig(p.configService, "auditdb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(auditDBManager))
	identityDBManager := identitydb.NewManager(cs, dbconfig.NewConfig(p.configService, "identitydb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(identityDBManager))

	// configure selector service
	var selectorManagerProvider token.SelectorManagerProvider
	switch cs.GetString("token.selector.driver") {
	case "simple":
		selectorManagerProvider = selector.NewProvider(
			network2.NewLockerProvider(ttxdbManager, 2*time.Second, 5*time.Minute),
			2,
			5*time.Second,
			tracing.Get(p.registry).GetTracer(),
		)
	case "mailman":
		// we use mailman as our default selector
		subscriber, err := events.GetSubscriber(p.registry)
		assert.NoError(err, "failed to get events subscriber")
		selectorManagerProvider = mailman.NewService(subscriber, tracing.Get(p.registry).GetTracer())
	default:
		tokenLockDBManager := tokenlockdb.NewManager(cs, dbconfig.NewConfig(p.configService, "tokenlockdb.persistence.type", "db.persistence.type"))
		assert.NoError(p.registry.RegisterService(tokenLockDBManager))
		selectorManagerProvider = sherdlock.NewService(tokenDBManager, tokenLockDBManager)
	}

	// Register the token management service provider
	tmsp := token.NewManagementServiceProvider(
		logging.MustGetLogger("token-sdk"),
		tmsProvider,
		networkProvider,
		&vault.ProviderAdaptor{Provider: networkProvider},
		network2.NewCertificationClientProvider(),
		selectorManagerProvider,
	)
	p.tmsProvider = tmsp
	assert.NoError(p.registry.RegisterService(tmsp))

	// Token Manager
	identityStorageProvider := identity.NewDBStorageProvider(kvs.GetService(p.registry), identityDBManager)
	assert.NoError(p.registry.RegisterService(identityStorageProvider), "failed to register identity storage")
	publisher, err := events.GetPublisher(p.registry)
	assert.NoError(err, "failed to get publisher")
	tokensManager := tokens.NewManager(tmsp, tokenDBManager, publisher, tokens2.NewAuthorizationMultiplexer(&tokens2.TMSAuthorization{}, &htlc.ScriptOwnership{}), tokens2.NewIssuedMultiplexer(&tokens2.WalletIssued{}))
	assert.NoError(p.registry.RegisterService(tokensManager))

	vaultProvider := vault.NewVaultProvider(tokenDBManager, ttxdbManager, auditDBManager)
	assert.NoError(p.registry.RegisterService(vaultProvider))

	ownerManager := ttx.NewManager(networkProvider, tmsp, ttxdbManager, tokensManager)
	assert.NoError(p.registry.RegisterService(ownerManager))
	auditorManager := auditor.NewManager(networkProvider, auditDBManager, tokensManager, tmsp)
	assert.NoError(p.registry.RegisterService(auditorManager))

	// configuration callback
	p.postInitializer, err = tms.NewPostInitializer(tmsp, tokensManager, networkProvider, ownerManager, auditorManager)
	assert.NoError(err)
	tmsProvider.SetCallback(p.postInitializer.PostInit)

	// Install metrics
	assert.NoError(
		p.registry.RegisterService(ttx.NewMetrics(metrics.GetProvider(p.registry))),
		"failed to register ttx package's metrics",
	)
	return nil
}

func (p *SDK) Start(ctx context.Context) error {
	if !p.configService.Enabled() {
		logger.Infof("Token platform not enabled, skipping start")
		return nil
	}
	logger.Infof("Token platform enabled, starting...")

	// load the configured tms
	configurations, err := p.configService.Configurations()
	if err != nil {
		return errors.WithMessagef(err, "failed get the configuration configurations")
	}
	logger.Infof("configured token management service [%d]", len(configurations))
	for _, tmsConfig := range configurations {
		tmsID := tmsConfig.ID()
		logger.Infof("start token management service [%s]...", tmsID)

		// connect network
		net, err := p.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
		if err != nil {
			return errors.Wrapf(err, "failed to get network [%s]", tmsID)
		}
		opts, err := net.Connect(tmsID.Namespace)
		if err != nil {
			return errors.WithMessagef(err, "failed to connect to connect backend to tms [%s]", tmsID)
		}
		_, err = p.tmsProvider.GetManagementService(opts...)
		if err != nil {
			return errors.WithMessagef(err, "failed to instantiate tms [%s]", tmsID)
		}
	}

	logger.Infof("Token platform enabled, starting...done")
	return nil
}
