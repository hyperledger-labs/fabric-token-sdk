/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	errors2 "errors"
	"time"

	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/dig"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	core2 "github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	dbconfig "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	logging2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/mailman"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/memory"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/memory"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	"github.com/pkg/errors"
	"go.uber.org/dig"
	_ "modernc.org/sqlite"
)

var logger = flogging.MustGetLogger("token-sdk")

type Registry interface {
	GetService(v interface{}) (interface{}, error)

	RegisterService(service interface{}) error
}

var selectorProviders = map[string]any{
	"simple":    selector.NewProvider,
	"mailman":   mailman.NewService,
	"sherdlock": sherdlock.NewService,
	"":          sherdlock.NewService,
}

type SDK struct {
	*fabricsdk.SDK
}

func (p *SDK) TokenEnabled() bool {
	return p.Config.GetBool("token.enabled")
}

func NewSDK(registry Registry) *SDK {
	return &SDK{SDK: fabricsdk.NewSDK(registry)}
}

func (p *SDK) Install() error {
	if !p.TokenEnabled() {
		logger.Infof("Token platform not enabled, skipping")
		return p.SDK.Install()
	}

	logger.Infof("Token platform enabled, installing...")

	logger.Infof("Set TMS TMSProvider")

	err := errors2.Join(
		p.C.Provide(func(sp view2.ServiceProvider) *network.Provider { return network.NewProvider(sp) }),
		p.C.Provide(digutils.Identity[*network.Provider](), dig.As(new(ttx.NetworkProvider), new(token.Normalizer), new(auditor.NetworkProvider))),
		p.C.Provide(func(networkProvider *network.Provider) *vault.PublicParamsProvider {
			return &vault.PublicParamsProvider{Provider: networkProvider}
		}, dig.As(new(core2.Vault))),
		p.C.Provide(digutils.Identity[driver.ConfigService](), dig.As(new(core.ConfigProvider))),
		p.C.Provide(func() logging2.Logger { return flogging.MustGetLogger("token-sdk.core") }),
		p.C.Provide(digutils.Identity[logging2.Logger](), dig.As(new(logging.Logger))),
		p.C.Provide(core2.NewTMSProvider),
		p.C.Provide(digutils.Identity[*core2.TMSProvider](), dig.As(new(driver2.TokenManagerServiceProvider))),
		p.C.Provide(func(service driver.ConfigService) *config2.Service { return config2.NewService(service) }),
		p.C.Provide(digutils.Identity[*config2.Service](), dig.As(new(core2.ConfigProvider))),
		p.C.Provide(func(ttxdbManager *ttxdb.Manager) *network2.LockerProvider {
			return network2.NewLockerProvider(ttxdbManager, 2*time.Second, 5*time.Minute)
		}),
		p.C.Provide(selectorProviders[p.Config.GetString("token.selector.driver")], dig.As(new(token.SelectorManagerProvider))),
		p.C.Provide(network2.NewCertificationClientProvider, dig.As(new(token.CertificationClientProvider))),
		p.C.Provide(func(networkProvider *network.Provider) *vault.ProviderAdaptor {
			return &vault.ProviderAdaptor{Provider: networkProvider}
		}, dig.As(new(token.VaultProvider))),
		p.C.Provide(token.NewManagementServiceProvider),
		p.C.Provide(digutils.Identity[*token.ManagementServiceProvider](), dig.As(new(ttx.TMSProvider), new(tokens.TMSProvider), new(auditor.TokenManagementServiceProvider))),
		p.C.Provide(func(configService driver.ConfigService, configProvider *config2.Service) *ttxdb.Manager {
			return ttxdb.NewManager(configService, dbconfig.NewConfig(configProvider, "ttxdb.persistence.type", "db.persistence.type"))
		}),
		p.C.Provide(digutils.Identity[*ttxdb.Manager](), dig.As(new(ttx.DBProvider), new(network2.TTXDBProvider))),
		p.C.Provide(func(configService driver.ConfigService, configProvider *config2.Service) *tokendb.Manager {
			return tokendb.NewManager(configService, dbconfig.NewConfig(configProvider, "tokendb.persistence.type", "db.persistence.type"))
		}),
		p.C.Provide(digutils.Identity[*tokendb.Manager](), dig.As(new(tokens.DBProvider))),
		p.C.Provide(func(configService driver.ConfigService, configProvider *config2.Service) *auditdb.Manager {
			return auditdb.NewManager(configService, dbconfig.NewConfig(configProvider, "ttxdb.persistence.type", "db.persistence.type"))
		}),
		p.C.Provide(digutils.Identity[*auditdb.Manager](), dig.As(new(auditor.AuditDBProvider))),
		p.C.Provide(func(configService driver.ConfigService, configProvider *config2.Service) *identitydb.Manager {
			return identitydb.NewManager(configService, dbconfig.NewConfig(configProvider, "ttxdb.persistence.type", "db.persistence.type"))
		}),
		p.C.Provide(func(configService driver.ConfigService, configProvider *config2.Service) *tokenlockdb.Manager {
			return tokenlockdb.NewManager(configService, dbconfig.NewConfig(configProvider, "tokenlockdb.persistence.type", "db.persistence.type"))
		}),
		p.C.Provide(digutils.Identity[*kvs.KVS](), dig.As(new(kvs2.KVS))),
		p.C.Provide(identity.NewDBStorageProvider),
		p.C.Provide(auditor.NewManager),
		p.C.Provide(ttx.NewManager),
		p.C.Provide(func() *tokens2.AuthorizationMultiplexer {
			return tokens2.NewAuthorizationMultiplexer(&tokens2.TMSAuthorization{}, &htlc.ScriptOwnership{})
		}, dig.As(new(tokens.Authorization))),
		p.C.Provide(func() *tokens2.IssuedMultiplexer { return tokens2.NewIssuedMultiplexer(&tokens2.WalletIssued{}) }, dig.As(new(tokens.Issued))),
		p.C.Provide(tokens.NewManager),
		p.C.Provide(digutils.Identity[*tokens.Manager](), dig.As(new(ttx.TokensProvider), new(auditor.TokenDBProvider))),
		p.C.Provide(vault.NewVaultProvider),
		p.C.Provide(tms.NewPostInitializer),
		p.C.Provide(ttx.NewMetrics),
	)

	if err != nil {
		return err
	}

	if err := p.SDK.Install(); err != nil {
		return err
	}

	// Backward compatibility with SP
	err = errors2.Join(
		digutils.Register[*network.Provider](p.C),
		digutils.Register[*token.ManagementServiceProvider](p.C),
		digutils.Register[*ttxdb.Manager](p.C),
		digutils.Register[*tokendb.Manager](p.C),
		digutils.Register[*auditdb.Manager](p.C),
		digutils.Register[*identitydb.Manager](p.C),
		digutils.Register[*vault.Provider](p.C),
		digutils.Register[driver.ConfigService](p.C),
		digutils.Register[*identity.DBStorageProvider](p.C),
		digutils.Register[*ttx.Metrics](p.C),
		digutils.Register[*auditor.Manager](p.C),
		digutils.Register[*config2.Service](p.C),
		digutils.Register[*ttx.Manager](p.C),
		digutils.Register[*tokens.Manager](p.C),
		digutils.Register[*tracing.Provider](p.C),
	)
	if err != nil {
		return err
	}

	return errors2.Join(
		p.C.Invoke(func(tmsProvider *core2.TMSProvider, postInitializer *tms.PostInitializer) {
			tmsProvider.SetCallback(postInitializer.PostInit)
		}),
	)
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

	return p.C.Invoke(func(configService *config2.Service, networkProvider *network.Provider, tmsProvider *token.ManagementServiceProvider) error {
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
			opts, err := net.Connect(tmsID.Namespace)
			if err != nil {
				return errors.WithMessagef(err, "failed to connect to connect backend to tms [%s]", tmsID)
			}
			_, err = tmsProvider.GetManagementService(opts...)
			if err != nil {
				return errors.WithMessagef(err, "failed to instantiate tms [%s]", tmsID)
			}
		}
		logger.Infof("Token platform enabled, starting...done")
		return nil
	})
}
