/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	tracing2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/observables"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

// Driver contains the non-static logic of the driver (including services)
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
		Name: fabtoken.PublicParameters,
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
	logger := logging.DriverLogger("token-sdk.driver.fabtoken", networkID, channel, namespace)

	if len(publicParams) == 0 {
		return nil, errors.Errorf("empty public parameters")
	}
	n, err := d.networkProvider.GetNetwork(networkID, channel)
	if err != nil {
		return nil, errors.Errorf("failed getting network [%s]", err)
	}
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}
	qe := v.QueryEngine()
	networkLocalMembership := n.LocalMembership()

	tmsConfig, err := d.configService.ConfigurationFor(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", networkID, channel, namespace)
	}

	publicParamsManager, err := common.NewPublicParamsManager[*fabtoken.PublicParams](
		&PublicParamsDeserializer{},
		fabtoken.PublicParameters,
		publicParams,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze public params manager")
	}

	ws, err := d.newWalletService(tmsConfig, d.endpointService, d.storageProvider, qe, logger, d.identityProvider.DefaultIdentity(), networkLocalMembership.DefaultIdentity(), nil, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze wallet service for [%s:%s]", networkID, namespace)
	}
	deserializer := ws.Deserializer
	ip := ws.IdentityProvider

	metricsProvider := metrics.NewTMSProvider(tmsConfig.ID(), d.metricsProvider)
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(d.tracerProvider, metricsProvider)

	service, err := fabtoken.NewService(
		logger,
		ws,
		publicParamsManager,
		ip,
		common.NewSerializer(),
		deserializer,
		tmsConfig,
		observables.NewObservableIssueService(
			fabtoken.NewIssueService(publicParamsManager, ws, deserializer),
			observables.NewIssue(tracerProvider),
		),
		observables.NewObservableTransferService(
			fabtoken.NewTransferService(logger, publicParamsManager, ws, common.NewVaultTokenLoader(qe), deserializer),
			observables.NewTransfer(tracerProvider),
		),
		observables.NewObservableAuditorService(
			fabtoken.NewAuditorService(),
			observables.NewAudit(tracerProvider),
		),
		fabtoken.NewTokensService(),
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}
	return service, nil
}

func (d *Driver) NewValidator(_ driver.ServiceProvider, tmsID driver.TMSID, params driver.PublicParameters) (driver.Validator, error) {

	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	metricsProvider := metrics.NewTMSProvider(tmsID, d.metricsProvider)
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(d.tracerProvider, metricsProvider)
	defaultValidator, err := d.DefaultValidator(pp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create default validator")
	}
	return observables.NewObservableValidator(defaultValidator, observables.NewValidator(tracerProvider)), nil
}
