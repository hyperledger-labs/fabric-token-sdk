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
	tms2 "github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/certification"
	dbconfig "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	tmsinit "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms/db"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/interactive"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/mailman"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/memory"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/memory"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	"github.com/pkg/errors"
	_ "modernc.org/sqlite"
)

var logger = flogging.MustGetLogger("token-sdk")

type Registry interface {
	GetService(v interface{}) (interface{}, error)

	RegisterService(service interface{}) error
}

type SDK struct {
	registry        Registry
	postInitializer *tmsinit.PostInitializer
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

	vaultProvider := db.NewVaultProvider(p.registry)
	assert.NoError(p.registry.RegisterService(vaultProvider))

	// Network provider
	networkProvider := network.NewProvider(p.registry)
	assert.NoError(p.registry.RegisterService(networkProvider))

	tmsProvider := tms2.NewTMSProvider(
		p.registry,
		flogging.MustGetLogger("token-sdk.core"),
		configProvider,
		&vault.PublicParamsProvider{Provider: networkProvider},
	)
	assert.NoError(p.registry.RegisterService(tmsProvider))

	// configure selector service
	var selectorManagerProvider token.SelectorManagerProvider
	switch configProvider.GetString("token.selector.driver") {
	case "simple":
		selectorManagerProvider = selector.NewProvider(
			network2.NewLockerProvider(p.registry, 2*time.Second, 5*time.Minute),
			2,
			5*time.Second,
			tracing.Get(p.registry).GetTracer(),
		)
	default:
		// we use mailman as our default selector
		subscriber, err := events.GetSubscriber(p.registry)
		assert.NoError(err, "failed to get events subscriber")
		selectorManagerProvider = mailman.NewService(
			tms.NewVaultProvider(networkProvider),
			subscriber,
			tracing.Get(p.registry).GetTracer(),
		)
	}

	// Register the token management service provider
	tmsp := token.NewManagementServiceProvider(
		tmsProvider,
		network2.NewNormalizer(config.NewTokenSDK(configProvider), p.registry),
		&vault.ProviderAdaptor{Provider: networkProvider},
		network2.NewCertificationClientProvider(),
		selectorManagerProvider,
	)
	assert.NoError(p.registry.RegisterService(tmsp))

	// DBs and their managers
	ttxdbManager := ttxdb.NewManager(p.registry, dbconfig.NewConfig(configProvider, "ttxdb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(ttxdbManager))
	tokenDBManager := tokendb.NewManager(p.registry, dbconfig.NewConfig(configProvider, "tokendb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(tokenDBManager))
	auditDBManager := auditdb.NewManager(p.registry, dbconfig.NewConfig(configProvider, "auditdb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(auditDBManager))
	identityDBManager := identitydb.NewManager(p.registry, dbconfig.NewConfig(configProvider, "identitydb.persistence.type", "db.persistence.type"))
	assert.NoError(p.registry.RegisterService(identityDBManager))
	identityStorageProvider := identity.NewDBStorageProvider(kvs.GetService(p.registry), identityDBManager)
	assert.NoError(p.registry.RegisterService(identityStorageProvider), "failed to register identity storage")
	auditorManager := auditor.NewManager(networkProvider, auditDBManager, storage.NewDBEntriesStorage("auditor", kvs.GetService(p.registry)))
	assert.NoError(p.registry.RegisterService(auditorManager))
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
	ownerManager := ttx.NewManager(networkProvider, tmsp, ttxdbManager, tokensManager, storage.NewDBEntriesStorage("owner", kvs.GetService(p.registry)))
	assert.NoError(p.registry.RegisterService(ownerManager))

	// TMS callback
	p.postInitializer = tmsinit.NewPostInitializer(p.registry, networkProvider, ownerManager, auditorManager)
	tmsProvider.SetCallback(p.postInitializer.PostInit)

	// Orion related initialization
	enabled, err := orion.IsCustodian(configProvider)
	assert.NoError(err, "failed to get custodian status")
	logger.Infof("Orion Custodian enabled: %t", enabled)
	if enabled {
		assert.NoError(orion.InstallViews(p.registry), "failed to install custodian views")
	}

	// Certification
	assert.NoError(p.registry.RegisterService(
		certification.NewDBStorageProvider(tokenDBManager)),
		"failed to register certification storage",
	)

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
		if err := p.postInitializer.ConnectNetwork(tmsID.Network, tmsID.Channel, tmsID.Namespace); err != nil {
			return errors.WithMessagef(err, "failed to connect to connect backend to tms [%s]", tmsID)
		}
	}

	logger.Infof("Token platform enabled, starting...done")
	return nil
}
