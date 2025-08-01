/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	v2 "github.com/hyperledger-labs/fabric-token-sdk/docs/core/extension/zkatdlog/nogh/v2/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/upgrade"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/multisig"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type Driver struct {
	*base
	metricsProvider  metrics.Provider
	tracerProvider   trace.TracerProvider
	configService    *config.Service
	storageProvider  identity.StorageProvider
	identityProvider endpoint.IdentityService
	endpointService  *endpoint.Service
	networkProvider  *network.Provider
	vaultProvider    *vault.Provider
}

func NewDriver(
	metricsProvider metrics.Provider,
	tracerProvider trace.TracerProvider,
	configService *config.Service,
	storageProvider identity.StorageProvider,
	identityProvider endpoint.IdentityService,
	endpointService *endpoint.Service,
	networkProvider *network.Provider,
	vaultProvider *vault.Provider,
) core.NamedFactory[driver.Driver] {
	return core.NamedFactory[driver.Driver]{
		Name: core.DriverIdentifier(v2.DLogNoGHDriverName, v2.ProtocolV2),
		Driver: &Driver{
			base:             &base{},
			metricsProvider:  metricsProvider,
			tracerProvider:   tracerProvider,
			configService:    configService,
			storageProvider:  storageProvider,
			identityProvider: identityProvider,
			endpointService:  endpointService,
			networkProvider:  networkProvider,
			vaultProvider:    vaultProvider,
		},
	}
}

func (d *Driver) NewTokenService(tmsID driver.TMSID, publicParams []byte) (driver.TokenManagerService, error) {
	logger := logging.DriverLogger("token-sdk.driver.zkatdlog", tmsID.Network, tmsID.Channel, tmsID.Namespace)

	logger.Debugf("creating new token service with public parameters [%s]", hash.Hashable(publicParams))

	if len(publicParams) == 0 {
		return nil, errors.Errorf("empty public parameters")
	}
	// get network
	n, err := d.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return nil, errors.Errorf("failed getting network [%s]", err)
	}

	// get vault
	vault, err := d.vaultProvider.Vault(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.Errorf("failed getting vault [%s]", err)
	}

	networkLocalMembership := n.LocalMembership()

	tmsConfig, err := d.configService.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID.Namespace)
	}

	ppm, err := common.NewPublicParamsManager[*setup.PublicParams](
		&PublicParamsDeserializer{},
		v2.DLogNoGHDriverName,
		v2.ProtocolV2,
		publicParams,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze public params manager")
	}

	pp := ppm.PublicParams(context.Background())
	logger.Infof("new token driver for tms id [%s] with label and version [%s:%d]: [%s]", tmsID, pp.TokenDriverName(), pp.TokenDriverVersion(), pp)

	metricsProvider := metrics.NewTMSProvider(tmsConfig.ID(), d.metricsProvider)
	qe := vault.QueryEngine()
	ws, err := d.newWalletService(
		tmsConfig,
		d.endpointService,
		d.storageProvider,
		qe,
		logger,
		d.identityProvider.DefaultIdentity(),
		networkLocalMembership.DefaultIdentity(),
		ppm.PublicParams(context.Background()),
		false,
		metricsProvider,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze wallet service for [%s:%s]", tmsID.Network, tmsID.Namespace)
	}
	deserializer := ws.Deserializer
	ip := ws.IdentityProvider

	authorization := common.NewAuthorizationMultiplexer(
		common.NewTMSAuthorization(logger, ppm.PublicParams(context.Background()), ws),
		htlc.NewScriptAuth(ws),
		multisig.NewEscrowAuth(ws),
	)

	driverMetrics := v1.NewMetrics(metricsProvider)
	tokensService, err := token3.NewTokensService(logger, ppm, deserializer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze token service for [%s:%s]", tmsID.Network, tmsID.Namespace)
	}
	tokensUpgradeService, err := upgrade.NewService(logger, ppm.PublicParams(context.Background()).QuantityPrecision, deserializer, ip)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze token upgrade service for [%s:%s]", tmsID.Network, tmsID.Namespace)
	}
	validator, err := validator.New(
		logger,
		ppm.PublicParams(context.Background()),
		deserializer,
		nil,
		nil,
		nil,
	), nil
	if err != nil {
		return nil, errors.Wrap(err, "failed to instantiate validator")
	}

	service, err := v1.NewTokenService(
		logger,
		ws,
		ppm,
		ip,
		deserializer,
		tmsConfig,
		v1.NewIssueService(logger, ppm, ws, deserializer, driverMetrics, tokensService, tokensUpgradeService),
		v1.NewTransferService(
			logger,
			ppm,
			ws,
			common.NewVaultLedgerTokenAndMetadataLoader[[]byte, []byte](qe, &common.IdentityTokenAndMetadataDeserializer{}),
			deserializer,
			driverMetrics,
			d.tracerProvider,
			tokensService,
		),
		v1.NewAuditorService(
			logger,
			ppm,
			common.NewLedgerTokenLoader[*token3.Token](logger, d.tracerProvider, qe, &TokenDeserializer{}),
			deserializer,
			driverMetrics,
			d.tracerProvider,
		),
		tokensService,
		tokensUpgradeService,
		authorization,
		validator,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}

	return service, err
}

func (d *Driver) NewDefaultValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*setup.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	return d.DefaultValidator(pp)
}
