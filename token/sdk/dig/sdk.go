/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	errors2 "errors"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	fscconfig "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	ftscore "github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	ftsdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	ftsconfig "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	sdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	db2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/keystoredb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/walletdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	auditor2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/auditor"
	wrapper2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/wrapper"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/dig"
)

var logger = logging.MustGetLogger()

var selectorProviders = map[sdriver.Driver]any{
	sdriver.Simple:    simple.NewService,
	sdriver.Sherdlock: sherdlock.NewService,
	"":                sherdlock.NewService,
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

		// config service
		p.Container().Provide(
			digutils.Identity[*fscconfig.Provider](),
			dig.As(new(ftsconfig.Provider), new(sherdlock.ConfigProvider), new(simple.ConfigProvider)),
		),
		p.Container().Provide(ftsconfig.NewService),
		p.Container().Provide(
			tms.NewConfigServiceWrapper,
			dig.As(new(ftscore.ConfigService), new(db2.ConfigService), new(tms.ConfigService)),
		),

		// network service
		p.Container().Provide(network.NewProvider),
		p.Container().Provide(newTokenDriverService),
		p.Container().Provide(
			digutils.Identity[*network.Provider](),
			dig.As(
				new(token.Normalizer),
				new(auditor.NetworkProvider),
				new(common2.NetworkProvider),
				new(tokens.NetworkProvider),
			),
		),
		p.Container().Provide(func(vaultProvider *vault.Provider) *vault.PublicParamsStorage {
			return &vault.PublicParamsStorage{Provider: vaultProvider}
		}, dig.As(new(ftscore.PublicParametersStorage))),

		p.Container().Provide(ftscore.NewTMSProvider),
		p.Container().Provide(digutils.Identity[*ftscore.TMSProvider](), dig.As(new(ftsdriver.TokenManagerServiceProvider))),
		p.Container().Provide(tms.NewPostInitializer),

		p.Container().Provide(func(ttxStoreServiceManager ttxdb.StoreServiceManager) *network2.LockerProvider {
			return network2.NewLockerProvider(ttxStoreServiceManager, 2*time.Second, 5*time.Minute)
		}, dig.As(new(simple.LockerProvider))),
		p.Container().Provide(selectorProviders[sdriver.Driver(p.ConfigService().GetString("token.selector.driver"))], dig.As(new(token.SelectorManagerProvider))),
		p.Container().Provide(network2.NewCertificationClientProvider, dig.As(new(token.CertificationClientProvider))),
		p.Container().Provide(token.NewManagementServiceProvider),
		p.Container().Provide(tms.NewTMSNormalizer, dig.As(new(token.TMSNormalizer))),
		p.Container().Provide(
			digutils.Identity[*token.ManagementServiceProvider](),
			dig.As(
				new(tokens.TMSProvider),
				new(auditor.TokenManagementServiceProvider),
				new(common2.TokenManagementServiceProvider),
			),
		),

		// selector service
		p.Container().Provide(func(tokenStoreServiceManager tokendb.StoreServiceManager, notifierManager tokendb.NotifierManager, metricsProvider metrics.Provider) sherdlock.FetcherProvider {
			return sherdlock.NewFetcherProvider(tokenStoreServiceManager, notifierManager, metricsProvider, sherdlock.Mixed)
		}),

		// storage
		p.Container().Provide(ttxdb.NewStoreServiceManager),
		p.Container().Provide(tokendb.NewStoreServiceManager),
		p.Container().Provide(tokendb.NewNotifierManager),
		p.Container().Provide(auditdb.NewStoreServiceManager),
		p.Container().Provide(identitydb.NewStoreServiceManager),
		p.Container().Provide(keystoredb.NewStoreServiceManager),
		p.Container().Provide(walletdb.NewStoreServiceManager),
		p.Container().Provide(tokenlockdb.NewStoreServiceManager),
		p.Container().Provide(identity.NewDBStorageProvider),
		p.Container().Provide(auditor.NewServiceManager),
		p.Container().Provide(tokens.NewServiceManager),
		p.Container().Provide(digutils.Identity[*tokens.ServiceManager](), dig.As(new(auditor.TokensServiceManager))),
		p.Container().Provide(vault.NewVaultProvider),
		p.Container().Provide(digutils.Identity[*vault.Provider](), dig.As(new(token.VaultProvider))),
		p.Container().Provide(sqlite.NewNamedDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(postgres.NewNamedDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(memory.NewNamedDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(newMultiplexedDriver),
		p.Container().Provide(NewAuditorCheckServiceProvider),
		p.Container().Provide(digutils.Identity[*db.AuditorCheckServiceProvider](), dig.As(new(auditor.CheckServiceProvider))),
		p.Container().Provide(NewOwnerCheckServiceProvider),

		// ttx service
		p.Container().Provide(wrapper2.NewTokenManagementServiceProvider, dig.As(new(dep.TokenManagementServiceProvider))),
		p.Container().Provide(wrapper2.NewAuditServiceProvider, dig.As(new(auditor2.ServiceProvider))),
		p.Container().Provide(wrapper2.NewNetworkProvider, dig.As(new(dep.NetworkProvider))),
		p.Container().Provide(wrapper2.NewNetworkIdentityProvider),
		p.Container().Provide(digutils.Identity[*wrapper2.NetworkIdentityProvider](), dig.As(new(dep.NetworkIdentityProvider))),
		p.Container().Provide(wrapper2.NewTransactionDBProvider, dig.As(new(dep.TransactionDBProvider))),
		p.Container().Provide(wrapper2.NewAuditDBProvider, dig.As(new(dep.AuditDBProvider))),
		p.Container().Provide(wrapper2.NewStorageProvider),
		p.Container().Provide(digutils.Identity[*tokens.ServiceManager](), dig.As(new(ttx.TokensServiceManager))),
		p.Container().Provide(digutils.Identity[*db.OwnerCheckServiceProvider](), dig.As(new(ttx.CheckServiceProvider))),
		p.Container().Provide(ttx.NewServiceManager),
		p.Container().Provide(ttx.NewMetrics),
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

	// Backward compatibility with SP
	err = errors2.Join(
		digutils.Register[*kvs.KVS](p.Container()),
		digutils.Register[*ftscore.TokenDriverService](p.Container()),
		digutils.Register[*network.Provider](p.Container()),
		digutils.Register[*token.ManagementServiceProvider](p.Container()),
		digutils.Register[ttxdb.StoreServiceManager](p.Container()),
		digutils.Register[tokendb.StoreServiceManager](p.Container()),
		digutils.Register[auditdb.StoreServiceManager](p.Container()),
		digutils.Register[identitydb.StoreServiceManager](p.Container()),
		digutils.Register[keystoredb.StoreServiceManager](p.Container()),
		digutils.Register[*vault.Provider](p.Container()),
		digutils.Register[driver.ConfigService](p.Container()),
		digutils.Register[*identity.DBStorageProvider](p.Container()),
		digutils.Register[*ttx.Metrics](p.Container()),
		digutils.Register[*auditor.ServiceManager](p.Container()),
		digutils.Register[*ftsconfig.Service](p.Container()),
		digutils.Register[*ttx.ServiceManager](p.Container()),
		digutils.Register[*tokens.ServiceManager](p.Container()),
		digutils.Register[trace.TracerProvider](p.Container()),
		digutils.Register[metrics.Provider](p.Container()),
		digutils.Register[dep.TokenManagementServiceProvider](p.Container()),
		digutils.Register[dep.NetworkProvider](p.Container()),
		digutils.Register[*wrapper2.StorageProvider](p.Container()),
		digutils.Register[*wrapper2.NetworkIdentityProvider](p.Container()),
		digutils.Register[dep.TransactionDBProvider](p.Container()),
		digutils.Register[dep.AuditDBProvider](p.Container()),
		digutils.Register[auditor2.ServiceProvider](p.Container()),
	)
	if err != nil {
		return errors.WithMessagef(err, "failed setting backward comaptibility with SP")
	}

	err = errors2.Join(
		p.Container().Invoke(func(tmsProvider *ftscore.TMSProvider, postInitializer *tms.PostInitializer) {
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

	if err := errors2.Join(
		p.Container().Invoke(registerNetworkDrivers),
		p.Container().Invoke(connectNetworks),
	); err != nil {
		logger.Errorf("Token platform enabled, starting...failed with error [%s]", err)
		return err
	}
	logger.Infof("Token platform enabled, starting...done")
	return nil
}

func connectNetworks(networkProvider *network.Provider) error {
	return networkProvider.Connect()
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
