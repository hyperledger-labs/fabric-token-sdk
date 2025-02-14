/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type Driver struct {
	*base
	metricsProvider  metrics.Provider
	tracerProvider   trace.TracerProvider
	configService    *config.Service
	storageProvider  identity.StorageProvider
	identityProvider view2.IdentityProvider
	endpointService  *view.EndpointService
	networkProvider  *network.Provider
}

func NewDriver(
	metricsProvider metrics.Provider,
	tracerProvider trace.TracerProvider,
	configService *config.Service,
	storageProvider identity.StorageProvider,
	identityProvider view2.IdentityProvider,
	endpointService *view.EndpointService,
	networkProvider *network.Provider,
) core.NamedFactory[driver.Driver] {
	return core.NamedFactory[driver.Driver]{
		Name: v1.DLogPublicParameters,
		Driver: &Driver{
			base:             &base{},
			metricsProvider:  metricsProvider,
			tracerProvider:   tracerProvider,
			configService:    configService,
			storageProvider:  storageProvider,
			identityProvider: identityProvider,
			endpointService:  endpointService,
			networkProvider:  networkProvider,
		},
	}
}

func (d *Driver) NewTokenService(tmsID driver.TMSID, publicParams []byte) (driver.TokenManagerService, error) {
	logger := logging.DriverLogger("token-sdk.driver.zkatdlog", tmsID.Network, tmsID.Channel, tmsID.Namespace)

	logger.Debugf("creating new token service with public parameters [%s]", hash.Hashable(publicParams))

	if len(publicParams) == 0 {
		return nil, errors.Errorf("empty public parameters")
	}
	n, err := d.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get network [%s]", tmsID.Network)
	}
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", tmsID.Network)
	}
	networkLocalMembership := n.LocalMembership()
	v, err := n.TokenVault(tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", tmsID.Network, tmsID.Namespace)
	}

	tmsConfig, err := d.configService.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID.Namespace)
	}

	ppm, err := common.NewPublicParamsManager[*v1.PublicParams](
		&PublicParamsDeserializer{},
		v1.DLogPublicParameters,
		publicParams,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze public params manager")
	}

	qe := v.QueryEngine()
	ws, err := d.newWalletService(tmsConfig, d.endpointService, d.storageProvider, qe, logger, d.identityProvider.DefaultIdentity(), networkLocalMembership.DefaultIdentity(), ppm.PublicParams(), false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze wallet service for [%s:%s]", tmsID.Network, tmsID.Namespace)
	}
	deserializer := ws.Deserializer
	ip := ws.IdentityProvider

	authorization := common.NewAuthorizationMultiplexer(
		common.NewTMSAuthorization(logger, ppm.PublicParams(), ws),
		htlc.NewScriptAuth(ws),
	)

	metricsProvider := metrics.NewTMSProvider(tmsConfig.ID(), d.metricsProvider)
	driverMetrics := zkatdlog.NewMetrics(metricsProvider)
	tokensService, err := zkatdlog.NewTokensService(ppm, deserializer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze token service for [%s:%s]", tmsID.Network, tmsID.Namespace)
	}
	service, err := zkatdlog.NewTokenService(
		logger,
		ws,
		ppm,
		ip,
		deserializer,
		tmsConfig,
		zkatdlog.NewIssueService(logger, ppm, ws, deserializer, driverMetrics, tokensService),
		zkatdlog.NewTransferService(
			logger,
			ppm,
			ws,
			common.NewVaultLedgerTokenAndMetadataLoader[[]byte, []byte](qe, &common.IdentityTokenAndMetadataDeserializer{}),
			deserializer,
			driverMetrics,
			d.tracerProvider,
			tokensService,
		),
		zkatdlog.NewAuditorService(
			logger,
			ppm,
			common.NewLedgerTokenLoader[*token3.Token](logger, d.tracerProvider, qe, &TokenDeserializer{}),
			deserializer,
			driverMetrics,
			d.tracerProvider,
		),
		tokensService,
		authorization,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}

	return service, err
}

func (d *Driver) NewDefaultValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*v1.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	return d.DefaultValidator(pp)
}
