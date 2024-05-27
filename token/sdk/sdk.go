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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	dbconfig "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
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

	// Network provider
	networkProvider := network.NewProvider(p.registry)
	assert.NoError(p.registry.RegisterService(networkProvider))

	tmsProvider := core.NewTMSProvider(
		p.registry,
		flogging.MustGetLogger("token-sdk.core"),
		configProvider,
		&vault.PublicParamsProvider{Provider: networkProvider},
	)
	assert.NoError(p.registry.RegisterService(tmsProvider))

	// DB Managers
	ttxdbManager := ttxdb.NewManager(configProvider, dbconfig.NewConfig(configProvider, "ttxdb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(ttxdbManager))
	tokenDBManager := tokendb.NewManager(configProvider, dbconfig.NewConfig(configProvider, "tokendb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(tokenDBManager))
	auditDBManager := auditdb.NewManager(configProvider, dbconfig.NewConfig(configProvider, "auditdb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(auditDBManager))
	identityDBManager := identitydb.NewManager(configProvider, dbconfig.NewConfig(configProvider, "identitydb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(identityDBManager))

	// configure selector service
	var selectorManagerProvider token.SelectorManagerProvider
	switch configProvider.GetString("token.selector.driver") {
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
		tokenLockDBManager := tokenlockdb.NewManager(configProvider, dbconfig.NewConfig(configProvider, "tokenlockdb.persistence.type", "db.persistence.type"))
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
	assert.NoError(p.registry.RegisterService(tmsp))

	// Token Manager
	identityStorageProvider := identity.NewDBStorageProvider(kvs.GetService(p.registry), identityDBManager)
	assert.NoError(p.registry.RegisterService(identityStorageProvider), "failed to register identity storage")
	publisher, err := events.GetPublisher(p.registry)
	assert.NoError(err, "failed to get publisher")
	tokensManager := tokens.NewManager(
		tmsp,
		tokenDBManager,
		publisher,
		tokens2.NewAuthorizationMultiplexer(&tokens2.TMSAuthorization{}, &htlc.ScriptOwnership{}),
		tokens2.NewIssuedMultiplexer(&tokens2.WalletIssued{}),
		storage.NewDBEntriesStorage("tokens", kvs.GetService(p.registry)),
	)
	assert.NoError(p.registry.RegisterService(tokensManager))

	vaultProvider := vault.NewVaultProvider(tokenDBManager, ttxdbManager, auditDBManager)
	assert.NoError(p.registry.RegisterService(vaultProvider))

	ownerManager := ttx.NewManager(networkProvider, tmsp, ttxdbManager, tokensManager, storage.NewDBEntriesStorage("owner", kvs.GetService(p.registry)))
	assert.NoError(p.registry.RegisterService(ownerManager))
	auditorManager := auditor.NewManager(networkProvider, auditDBManager, tokensManager, storage.NewDBEntriesStorage("auditor", kvs.GetService(p.registry)), tmsp)
	assert.NoError(p.registry.RegisterService(auditorManager))

	// TMS callback
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
	//tmsProvider := token.GetManagementServiceProvider(p.registry)
	logger.Infof("configured token management service [%d]", len(tmsConfigs))
	for _, tmsConfig := range tmsConfigs {
		tmsID := token.TMSID{
			Network:   tmsConfig.TMS().Network,
			Channel:   tmsConfig.TMS().Channel,
			Namespace: tmsConfig.TMS().Namespace,
		}
		logger.Infof("start token management service [%s]...", tmsID)

		// connect network
		net := network.GetInstance(p.registry, tmsID.Network, tmsID.Channel)
		if net == nil {
			return errors.Wrapf(err, "failed to get network [%s]", tmsID)
		}
		opts, err := net.Connect(tmsID.Namespace)
		if err != nil {
			return errors.WithMessagef(err, "failed to connect to connect backend to tms [%s]", tmsID)
		}
		tms := token.GetManagementService(p.registry, opts...)
		if tms == nil {
			return errors.WithMessagef(err, "failed to instantiate tms [%s]", tmsID)
		}
	}

	logger.Infof("Token platform enabled, starting...done")
	return nil
}
