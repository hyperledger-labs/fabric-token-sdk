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
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	auditdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	identity2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	identitydriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	sdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	tokenlockdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	ttxdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
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
		p.Container().Provide(digutils.Identity[*network.Provider](), dig.As(new(ttx.NetworkProvider), new(token.Normalizer), new(auditor.NetworkProvider))),
		p.Container().Provide(func(networkProvider *network.Provider) *vault.PublicParamsProvider {
			return &vault.PublicParamsProvider{Provider: networkProvider}
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
		p.Container().Provide(func(networkProvider *network.Provider) *vault.ProviderAdaptor {
			return &vault.ProviderAdaptor{Provider: networkProvider}
		}, dig.As(new(token.VaultProvider))),
		p.Container().Provide(token.NewManagementServiceProvider),
		p.Container().Provide(token.NewTMSNormalizer, dig.As(new(token.TMSNormalizer))),
		p.Container().Provide(digutils.Identity[*token.ManagementServiceProvider](), dig.As(new(ttx.TMSProvider), new(tokens.TMSProvider), new(auditor.TokenManagementServiceProvider))),
		p.Container().Provide(NewTTXDBManager),
		p.Container().Provide(digutils.Identity[*ttxdb.Manager](), dig.As(new(ttx.DBProvider), new(network2.TTXDBProvider))),
		p.Container().Provide(NewTokenManagers),
		p.Container().Provide(digutils.Identity[*tokendb.Manager](), dig.As(new(tokens.DBProvider))),
		p.Container().Provide(NewAuditDBManager),
		p.Container().Provide(digutils.Identity[*auditdb.Manager](), dig.As(new(auditor.AuditDBProvider))),
		p.Container().Provide(NewIdentityDBManager),
		p.Container().Provide(NewTokenLockDBManager),
		p.Container().Provide(digutils.Identity[*kvs.KVS](), dig.As(new(kvs2.KVS))),
		p.Container().Provide(identity.NewDBStorageProvider),
		p.Container().Provide(digutils.Identity[*identity.DBStorageProvider](), dig.As(new(identity2.StorageProvider))),
		p.Container().Provide(auditor.NewManager),
		p.Container().Provide(ttx.NewManager),
		p.Container().Provide(tokens.NewManager),
		p.Container().Provide(digutils.Identity[*tokens.Manager](), dig.As(new(ttx.TokensProvider), new(auditor.TokenDBProvider))),
		p.Container().Provide(vault.NewVaultProvider),
		p.Container().Provide(tms.NewPostInitializer),
		p.Container().Provide(ttx.NewMetrics),
		p.Container().Provide(func(tracerProvider trace.TracerProvider) *tracing.TracerProvider {
			return tracing.NewTracerProvider(tracerProvider)
		}),
		p.Container().Provide(tokenlockdriver.NewDriver, dig.Group("tokenlockdb-drivers")),
		p.Container().Provide(auditdriver.NewDriver, dig.Group("auditdb-drivers")),
		p.Container().Provide(NewTokenDrivers),
		p.Container().Provide(ttxdriver.NewDriver, dig.Group("ttxdb-drivers")),
		p.Container().Provide(identitydriver.NewDriver, dig.Group("identitydb-drivers")),
		p.Container().Provide(NewDBDrivers),
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
		digutils.Register[*driver2.TokenDriverService](p.Container()),
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

func connectNetworks(configService *config2.Service, networkProvider *network.Provider, tmsProvider *token.ManagementServiceProvider) error {
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
	logger.Debugf("registering [%d] network drivers", len(in.Drivers))
	for _, d := range in.Drivers {
		in.NetworkProvider.RegisterDriver(d)
	}
}
