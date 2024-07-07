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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
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
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}

	tmsConfig, err := d.configService.ConfigurationFor(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", networkID, channel, namespace)
	}

	fscIdentity := d.identityProvider.DefaultIdentity()
	// Prepare roles
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
	ppm, err := common.NewPublicParamsManager[*crypto.PublicParams](
		&PublicParamsDeserializer{},
		crypto.DLogPublicParameters,
		publicParams,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze public params manager")
	}
	identityConfig, err := config2.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create identity config")
	}
	roleFactory := msp.NewRoleFactory(
		tmsID,
		identityConfig,                           // config
		fscIdentity,                              // FSC identity
		networkLocalMembership.DefaultIdentity(), // network default identity
		ip,
		sigService,        // signer service
		d.endpointService, // endpoint service
		d.storageProvider,
		deserializerManager,
		false,
	)
	role, err := roleFactory.NewIdemix(
		driver.OwnerRole,
		identityConfig.DefaultCacheSize(),
		ppm.PublicParams().IdemixIssuerPK,
		ppm.PublicParams().IdemixCurveID,
	)
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
	qe := v.QueryEngine()
	// wallet service
	walletDB, err := d.storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}

	deserializer, err := NewDeserializer(ppm.PublicParams())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate the deserializer")
	}
	ws := common.NewWalletService(
		logger,
		ip,
		deserializer,
		zkatdlog.NewWalletFactory(logger, ip, qe, identityConfig, deserializer),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)
	tokDeserializer := &TokenDeserializer{}

	metricsProvider := metrics.NewTMSProvider(tmsID, d.metricsProvider)
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(d.tracerProvider, metricsProvider)
	driverMetrics := zkatdlog.NewMetrics(metricsProvider)
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
				common.NewVaultLedgerTokenAndMetadataLoader[*token3.Token, *token3.Metadata](qe, tokDeserializer),
				deserializer,
				driverMetrics,
			),
			observables.NewTransfer(tracerProvider),
		),
		observables.NewObservableAuditorService(
			zkatdlog.NewAuditorService(
				logger,
				ppm,
				common.NewLedgerTokenLoader[*token3.Token](logger, qe, tokDeserializer),
				deserializer,
				driverMetrics,
			),
			observables.NewAudit(tracerProvider),
		),
		zkatdlog.NewTokensService(ppm),
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
