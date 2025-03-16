/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	errors2 "errors"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	core2 "github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/tracing"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	db2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/unity"
	identity2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	sdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/dig"
)

var logger = logging.MustGetLogger("token-sdk")

var selectorProviders = map[sdriver.Driver]any{
	sdriver.Simple:    selector.NewService,
	sdriver.Sherdlock: sherdlock.NewService,
	"":                sherdlock.NewService,
}

type SDK struct {
	dig2.SDK
}

func (p *SDK) TokenEnabled() bool {
	return p.ConfigService().GetBool("token.enabled")
}

func NewSDK(registry node.Registry) *SDK {
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
		p.Container().Provide(network.NewProvider),
		p.Container().Provide(newTokenDriverService),
		p.Container().Provide(
			digutils.Identity[*network.Provider](),
			dig.As(
				new(ttx.NetworkProvider),
				new(token.Normalizer),
				new(auditor.NetworkProvider),
				new(common2.NetworkProvider),
				new(tokens.NetworkProvider),
			),
		),
		p.Container().Provide(func(vaultProvider *vault.Provider) *vault.PublicParamsProvider {
			return &vault.PublicParamsProvider{Provider: vaultProvider}
		}, dig.As(new(core2.Vault))),
		p.Container().Provide(digutils.Identity[driver.ConfigService](), dig.As(new(core.ConfigProvider))),
		p.Container().Provide(func() logging.Logger { return logging.MustGetLogger("token-sdk.core") }),
		p.Container().Provide(core2.NewTMSProvider),
		p.Container().Provide(digutils.Identity[*core2.TMSProvider](), dig.As(new(driver2.TokenManagerServiceProvider))),
		p.Container().Provide(func(service driver.ConfigService) *config2.Service { return config2.NewService(service) }),
		p.Container().Provide(digutils.Identity[*config2.Service](), dig.As(new(core2.ConfigProvider))),
		p.Container().Provide(func(ttxdbManager *ttxdb.Manager) *network2.LockerProvider {
			return network2.NewLockerProvider(ttxdbManager, 2*time.Second, 5*time.Minute)
		}, dig.As(new(selector.LockerProvider))),
		p.Container().Provide(selectorProviders[sdriver.Driver(p.ConfigService().GetString("token.selector.driver"))], dig.As(new(token.SelectorManagerProvider))),
		p.Container().Provide(network2.NewCertificationClientProvider, dig.As(new(token.CertificationClientProvider))),
		p.Container().Provide(func(vaultProvider *vault.Provider) *vault.ProviderAdaptor {
			return &vault.ProviderAdaptor{Provider: vaultProvider}
		}, dig.As(new(token.VaultProvider))),
		p.Container().Provide(token.NewManagementServiceProvider),
		p.Container().Provide(token.NewTMSNormalizer, dig.As(new(token.TMSNormalizer))),
		p.Container().Provide(
			digutils.Identity[*token.ManagementServiceProvider](),
			dig.As(
				new(ttx.TMSProvider),
				new(tokens.TMSProvider),
				new(auditor.TokenManagementServiceProvider),
				new(common2.TokenManagementServiceProvider),
			),
		),
		p.Container().Provide(func(dh *db2.DriverHolder) *ttxdb.Manager {
			return ttxdb.NewManager(dh, "ttxdb.persistence", "db.persistence")
		}),
		p.Container().Provide(digutils.Identity[*ttxdb.Manager](), dig.As(new(ttx.DBProvider), new(network2.TTXDBProvider))),
		p.Container().Provide(func(dh *db2.DriverHolder) *tokendb.Manager {
			return tokendb.NewManager(dh, "tokendb.persistence", "db.persistence")
		}),
		p.Container().Provide(func(dh *db2.DriverHolder) *tokendb.NotifierManager {
			return tokendb.NewNotifierManager(dh, "tokendb.persistence", "db.persistence")
		}),
		p.Container().Provide(digutils.Identity[*tokendb.Manager](), dig.As(new(tokens.DBProvider))),
		p.Container().Provide(NewDriverHolder),
		p.Container().Provide(func(dh *db2.DriverHolder) *auditdb.Manager {
			return auditdb.NewManager(dh, "auditdb.persistence", "db.persistence")
		}),
		p.Container().Provide(digutils.Identity[*auditdb.Manager](), dig.As(new(auditor.AuditDBProvider))),
		p.Container().Provide(func(dh *db2.DriverHolder) *identitydb.Manager {
			return identitydb.NewManager(dh, "identitydb.persistence", "db.persistence")
		}),
		p.Container().Provide(func(dh *db2.DriverHolder) *tokenlockdb.Manager {
			return tokenlockdb.NewManager(dh, "tokenlockdb.persistence", "db.persistence")
		}),
		p.Container().Provide(digutils.Identity[*kvs.KVS](), dig.As(new(identity2.Keystore))),
		p.Container().Provide(identity.NewDBStorageProvider),
		p.Container().Provide(digutils.Identity[*identity.DBStorageProvider](), dig.As(new(identity2.StorageProvider))),
		p.Container().Provide(NewAuditorCheckServiceProvider),
		p.Container().Provide(digutils.Identity[*db.AuditorCheckServiceProvider](), dig.As(new(auditor.CheckServiceProvider))),
		p.Container().Provide(auditor.NewManager),
		p.Container().Provide(NewOwnerCheckServiceProvider),
		p.Container().Provide(digutils.Identity[*db.OwnerCheckServiceProvider](), dig.As(new(ttx.CheckServiceProvider))),
		p.Container().Provide(ttx.NewManager),
		p.Container().Provide(tokens.NewManager),
		p.Container().Provide(digutils.Identity[*tokens.Manager](), dig.As(new(ttx.TokensProvider), new(auditor.TokenDBProvider))),
		p.Container().Provide(vault.NewVaultProvider),
		p.Container().Provide(tms.NewPostInitializer),
		p.Container().Provide(ttx.NewMetrics),
		p.Container().Provide(func(tracerProvider trace.TracerProvider) *tracing.TracerProvider {
			return tracing.NewTracerProvider(tracerProvider)
		}),
		p.Container().Provide(unity.NewUnityDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(sql.NewDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(memory.NewDriver, dig.Group("token-db-drivers")),
		p.Container().Provide(func(dbManager *tokendb.Manager, notifierManager *tokendb.NotifierManager, metricsProvider metrics.Provider) sherdlock.FetcherProvider {
			return sherdlock.NewFetcherProvider(dbManager, notifierManager, metricsProvider, sherdlock.Mixed)
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

	// Backward compatibility with SP
	err = errors2.Join(
		digutils.Register[*kvs.KVS](p.Container()),
		digutils.Register[*core2.TokenDriverService](p.Container()),
		digutils.Register[*network.Provider](p.Container()),
		digutils.Register[*token.ManagementServiceProvider](p.Container()),
		digutils.Register[*ttxdb.Manager](p.Container()),
		digutils.Register[*tokendb.Manager](p.Container()),
		digutils.Register[*auditdb.Manager](p.Container()),
		digutils.Register[*identitydb.Manager](p.Container()),
		digutils.Register[*vault.Provider](p.Container()),
		digutils.Register[driver.ConfigService](p.Container()),
		digutils.Register[*identity.DBStorageProvider](p.Container()),
		digutils.Register[*ttx.Metrics](p.Container()),
		digutils.Register[*auditor.Manager](p.Container()),
		digutils.Register[*config2.Service](p.Container()),
		digutils.Register[*ttx.Manager](p.Container()),
		digutils.Register[*tokens.Manager](p.Container()),
		digutils.Register[trace.TracerProvider](p.Container()),
		digutils.Register[metrics.Provider](p.Container()),
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
