/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	tracing2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/observables"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
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
) driver.NamedFactory[driver.Driver] {
	return driver.NamedFactory[driver.Driver]{
		Name: crypto.DLogPublicParameters,
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

func (d *Driver) NewTokenService(_ driver.ServiceProvider, networkID string, channel string, namespace string, publicParams []byte) (driver.TokenManagerService, error) {
	logger := logging.DriverLogger("token-sdk.driver.zkatdlog", networkID, channel, namespace)

	logger.Debugf("creating new token service with public parameters [%s]", hash.Hashable(publicParams))

	if len(publicParams) == 0 {
		return nil, errors.Errorf("empty public parameters")
	}
	n, err := d.networkProvider.GetNetwork(networkID, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get network [%s]", networkID)
	}
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	networkLocalMembership := n.LocalMembership()
	v, err := n.TokenVault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}

	tmsConfig, err := d.configService.ConfigurationFor(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", networkID, channel, namespace)
	}

	ppm, err := common.NewPublicParamsManager[*crypto.PublicParams](
		&PublicParamsDeserializer{},
		crypto.DLogPublicParameters,
		publicParams,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze public params manager")
	}

	qe := v.QueryEngine()
	ws, err := d.newWalletService(tmsConfig, d.endpointService, d.storageProvider, qe, logger, d.identityProvider.DefaultIdentity(), networkLocalMembership.DefaultIdentity(), ppm.PublicParams(), false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze wallet service for [%s:%s]", networkID, namespace)
	}
	deserializer := ws.Deserializer
	ip := ws.IdentityProvider

	authorization := common.NewAuthorizationMultiplexer(
		common.NewTMSAuthorization(logger, ppm.PublicParams(), ws),
		htlc.NewScriptAuth(ws),
	)

	metricsProvider := metrics.NewTMSProvider(tmsConfig.ID(), d.metricsProvider)
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(d.tracerProvider, metricsProvider)
	driverMetrics := zkatdlog.NewMetrics(metricsProvider)
	tokensService, err := zkatdlog.NewTokensService(ppm)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze token service for [%s:%s]", networkID, namespace)
	}
	service, err := zkatdlog.NewTokenService(
		logger,
		ws,
		ppm,
		ip,
		common.NewSerializer(),
		deserializer,
		tmsConfig,
		observables.NewObservableIssueService(
			zkatdlog.NewIssueService(ppm, ws, deserializer, driverMetrics),
			observables.NewIssue(tracerProvider),
		),
		observables.NewObservableTransferService(
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
			observables.NewTransfer(tracerProvider),
		),
		observables.NewObservableAuditorService(
			zkatdlog.NewAuditorService(
				logger,
				ppm,
				common.NewLedgerTokenLoader[*token3.Token](logger, d.tracerProvider, qe, &TokenDeserializer{}),
				deserializer,
				driverMetrics,
				d.tracerProvider,
			),
			observables.NewAudit(tracerProvider),
		),
		tokensService,
		authorization,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}

	return service, err
}

func (d *Driver) NewValidator(_ driver.ServiceProvider, tmsID driver.TMSID, params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	defaultValidator, err := d.DefaultValidator(pp)
	if err != nil {
		return nil, err
	}
	metricsProvider := metrics.NewTMSProvider(tmsID, d.metricsProvider)
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(d.tracerProvider, metricsProvider)
	return observables.NewObservableValidator(defaultValidator, observables.NewValidator(tracerProvider)), nil
}
