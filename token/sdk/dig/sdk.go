/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	errors2 "errors"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	core2 "github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/support"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/support/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/support/identity"
	network3 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/support/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/support/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/support/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	identity2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	driver4 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	dbcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/walletdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	sdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/selector/driver"
	sherdlock2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/selector/sherdlock"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/logging"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

var logger = logging.MustGetLogger()

var selectorProviders = map[sdriver.Driver]any{
	sdriver.Simple:    selector.NewService,
	sdriver.Sherdlock: sherdlock2.NewService,
	"":                sherdlock2.NewService,
}

type SDK struct {
	dig2.SDK
}

func (p *SDK) TokenEnabled() bool {
	return p.ConfigService().GetBool("token.enabled")
}

func NewSDK(registry services.Registry) *SDK {
	return NewFrom(fabricsdk.NewSDK(registry))
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	if !p.TokenEnabled() {
		logger.Infof("Token platform not enabled, skipping")
		return p.SDK.Install()
	}

	logger.Infof("Token platform enabled, installing...")
	err := errors2.Join(
		p.Container().Provide(common.NewAcceptTxInDBFilterProvider),

		// network
		p.Container().Provide(network.NewProvider),
		p.Container().Provide(
			digutils.Identity[*network.Provider](),
			dig.As(
				new(ttx.NetworkProvider),
				new(token.Normalizer),
				new(auditor.NetworkProvider),
				new(dbcommon.NetworkProvider),
				new(tokens.NetworkProvider),
			),
		),
		p.Container().Provide(token.NewTMSNormalizer, dig.As(new(token.TMSNormalizer))),

		// token drivers
		p.Container().Provide(support.NewTokenDriverService),
		p.Container().Provide(func(vaultProvider *vault.Provider) *vault.PublicParamsStorage {
			return &vault.PublicParamsStorage{Provider: vaultProvider}
		}, dig.As(new(core2.PublicParametersStorage))),
		p.Container().Provide(vault.NewProvider),

		p.Container().Provide(digutils.Identity[driver.ConfigService](), dig.As(new(core.ConfigProvider))),
		p.Container().Provide(func() logging.Logger { return logging.MustGetLogger() }),
		p.Container().Provide(core2.NewTMSProvider),
		p.Container().Provide(digutils.Identity[*core2.TMSProvider](), dig.As(new(driver2.TokenManagerServiceProvider))),
		p.Container().Provide(func(service driver.ConfigService) *config2.Service { return config2.NewService(service) }),
		p.Container().Provide(digutils.Identity[*config2.Service](), dig.As(new(core2.ConfigProvider))),
		p.Container().Provide(func(ttxStoreServiceManager ttxdb.StoreServiceManager) *network3.LockerProvider {
			return network3.NewLockerProvider(ttxStoreServiceManager, 2*time.Second, 5*time.Minute)
		}, dig.As(new(selector.LockerProvider))),
		p.Container().Provide(selectorProviders[sdriver.Driver(p.ConfigService().GetString("token.selector.driver"))], dig.As(new(token.SelectorManagerProvider))),
		p.Container().Provide(network3.NewCertificationClientProvider, dig.As(new(token.CertificationClientProvider))),
		p.Container().Provide(token.NewManagementServiceProvider),
		p.Container().Provide(
			digutils.Identity[*token.ManagementServiceProvider](),
			dig.As(
				new(ttx.TMSProvider),
				new(tokens.TMSProvider),
				new(auditor.TokenManagementServiceProvider),
				new(dbcommon.TokenManagementServiceProvider),
			),
		),

		// storage
		p.Container().Provide(support.NewDBMultiplexedDriver),
		p.Container().Provide(ttxdb.NewStoreServiceManager),
		p.Container().Provide(digutils.Identity[ttxdb.StoreServiceManager](), dig.As(new(ttx.StoreServiceManager))),
		p.Container().Provide(tokendb.NewStoreServiceManager),
		p.Container().Provide(tokendb.NewNotifierManager),
		p.Container().Provide(digutils.Identity[tokendb.StoreServiceManager](), dig.As(new(tokens.StoreServiceManager))),
		p.Container().Provide(auditdb.NewStoreServiceManager),
		p.Container().Provide(digutils.Identity[auditdb.StoreServiceManager](), dig.As(new(auditor.StoreServiceManager))),
		p.Container().Provide(identitydb.NewStoreServiceManager),
		p.Container().Provide(walletdb.NewStoreServiceManager),
		p.Container().Provide(tokenlockdb.NewStoreServiceManager),
		p.Container().Provide(func(kvs *kvs.KVS) driver4.Keystore { return kvs2.Keystore(kvs) }),
		p.Container().Provide(identity.NewStorageProvider),
		p.Container().Provide(digutils.Identity[*identity.StorageProvider](), dig.As(new(identity2.StorageProvider))),
		p.Container().Provide(sqlite.NewNamedDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(postgres.NewNamedDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(memory.NewNamedDriver, dig.Group("token-db-drivers")),

		// auditor
		p.Container().Provide(support.NewAuditorCheckServiceProvider),
		p.Container().Provide(digutils.Identity[*db.AuditorCheckServiceProvider](), dig.As(new(auditor.CheckServiceProvider))),
		p.Container().Provide(auditor.NewServiceManager),

		// owner
		p.Container().Provide(support.NewOwnerCheckServiceProvider),
		p.Container().Provide(digutils.Identity[*db.OwnerCheckServiceProvider](), dig.As(new(ttx.CheckServiceProvider))),

		// tokens
		p.Container().Provide(tokens.NewServiceManager),
		p.Container().Provide(digutils.Identity[*tokens.ServiceManager](), dig.As(new(ttx.TokensServiceManager), new(auditor.TokensServiceManager))),

		p.Container().Provide(ttx.NewServiceManager),
		p.Container().Provide(digutils.Identity[*vault.Provider](), dig.As(new(token.VaultProvider))),
		p.Container().Provide(tms.NewPostInitializer),
		p.Container().Provide(ttx.NewMetrics),
		p.Container().Provide(func(tokenStoreServiceManager tokendb.StoreServiceManager, notifierManager tokendb.NotifierManager, metricsProvider metrics.Provider) sherdlock2.FetcherProvider {
			return sherdlock2.NewFetcherProvider(tokenStoreServiceManager, notifierManager, metricsProvider, sherdlock2.Mixed)
		}),
	)
	if err != nil {
		return errors.WithMessagef(err, "failed setting up dig container")
	}

	// Overwrite dependencies
	err = p.Container().Decorate(func(committer.DependencyResolver) committer.DependencyResolver {
		return committer.NewParallelDependencyResolver()
	})
	if err != nil {
		return errors.WithMessagef(err, "failed setting up decorator")
	}

	if err := p.SDK.Install(); err != nil {
		return errors.WithMessagef(err, "failed installing dig chain")
	}

	// Register services in services.Registry
	err = errors2.Join(
		// core
		digutils.Register[*core2.TokenDriverService](p.Container()),
		digutils.Register[*token.ManagementServiceProvider](p.Container()),
		digutils.Register[driver.ConfigService](p.Container()),

		// storage
		digutils.Register[*kvs.KVS](p.Container()),
		digutils.Register[ttxdb.StoreServiceManager](p.Container()),
		digutils.Register[tokendb.StoreServiceManager](p.Container()),
		digutils.Register[auditdb.StoreServiceManager](p.Container()),
		digutils.Register[identitydb.StoreServiceManager](p.Container()),
		digutils.Register[*identity.StorageProvider](p.Container()),

		// services
		digutils.Register[*network.Provider](p.Container()),
		digutils.Register[*ttx.Metrics](p.Container()),
		digutils.Register[*auditor.ServiceManager](p.Container()),
		digutils.Register[*ttx.ServiceManager](p.Container()),
		digutils.Register[*tokens.ServiceManager](p.Container()),
	)
	if err != nil {
		return errors.WithMessagef(err, "failed setting backward comaptibility with SP")
	}

	err = errors2.Join(
		p.Container().Invoke(func(tmsProvider *core2.TMSProvider, postInitializer *tms.PostInitializer) {
			tmsProvider.SetCallback(postInitializer.PostInit)
		}),
	)
	if err != nil {
		return errors.WithMessagef(err, "failed post-inititialization")
	}
	return nil
}

func (p *SDK) Start(ctx context.Context) error {
	if err := p.SDK.Start(ctx); err != nil {
		return err
	}
	if !p.TokenEnabled() {
		logger.Infof("Token platform not enabled, skipping start")
		return nil
	}
	logger.Infof("Token platform enabled, starting...")

	return errors2.Join(
		p.Container().Invoke(registerNetworkDrivers),
		p.Container().Invoke(connectNetworks),
	)
}

func connectNetworks(configService *config2.Service, networkProvider *network.Provider, _ *token.ManagementServiceProvider) error {
	configurations, err := configService.Configurations()
	if err != nil {
		return err
	}
	for _, tmsConfig := range configurations {
		tmsID := tmsConfig.ID()
		logger.Infof("start token management service [%s]...", tmsID)

		// connect network
		net, err := networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
		if err != nil {
			return errors.Wrapf(err, "failed to get network [%s]", tmsID)
		}
		_, err = net.Connect(tmsID.Namespace)
		if err != nil {
			return errors.WithMessagef(err, "failed to connect to connect backend to tms [%s]", tmsID)
		}
	}
	logger.Infof("Token platform enabled, starting...done")
	return nil
}

func registerNetworkDrivers(in struct {
	dig.In
	NetworkProvider *network.Provider
	Drivers         []driver3.Driver `group:"network-drivers"`
}) {
	for _, d := range in.Drivers {
		in.NetworkProvider.RegisterDriver(d)
	}
}
