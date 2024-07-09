/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	tracing2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/observables"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
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
	v, err := n.TokenVault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}
	qe := v.QueryEngine()
	networkLocalMembership := n.LocalMembership()

	tmsConfig, err := d.configService.ConfigurationFor(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", networkID, channel, namespace)
	}

	// Prepare roles
	fscIdentity := d.identityProvider.DefaultIdentity()
	roles := identity.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	tmsID := token.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	identityDB, err := d.storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(identityDB, sigService, d.endpointService, NewEIDRHDeserializer(), deserializerManager)
	identityConfig, err := config2.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create identity config")
	}
	roleFactory := msp.NewRoleFactory(
		tmsID,
		identityConfig,                           //config
		fscIdentity,                              // FSC identity
		networkLocalMembership.DefaultIdentity(), // network default identity
		ip,
		sigService,        // sig service
		d.endpointService, // endpoint service
		d.storageProvider,
		deserializerManager,
		false,
	)
	role, err := roleFactory.NewWrappedX509(driver.OwnerRole, false)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles.Register(driver.OwnerRole, role)
	role, err = roleFactory.NewX509(driver.IssuerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(driver.IssuerRole, role)
	role, err = roleFactory.NewX509(driver.AuditorRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor role")
	}
	roles.Register(driver.AuditorRole, role)
	role, err = roleFactory.NewX509(driver.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier role")
	}
	roles.Register(driver.CertifierRole, role)

	// Instantiate the token service
	walletDB, err := d.storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	publicParamsManager, err := common.NewPublicParamsManager[*fabtoken.PublicParams](
		&PublicParamsDeserializer{},
		fabtoken.PublicParameters,
		publicParams,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze public params manager")
	}
	deserializer := NewDeserializer()
	metricsProvider := metrics.NewTMSProvider(tmsID, d.metricsProvider)
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(d.tracerProvider, metricsProvider)
	ws := common.NewWalletService(
		logger,
		ip,
		deserializer,
		fabtoken.NewWalletFactory(logger, ip, qe),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)
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
