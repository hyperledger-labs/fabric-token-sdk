/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	tracing2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/observables"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

type Driver struct {
}

func (d *Driver) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := crypto.NewPublicParamsFromBytes(params, crypto.DLogPublicParameters)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *Driver) NewTokenService(sp driver.ServiceProvider, networkID string, channel string, namespace string, publicParams []byte) (driver.TokenManagerService, error) {
	logger := logging.DriverLogger("token-sdk.driver.zkatdlog", networkID, channel, namespace)

	if len(publicParams) == 0 {
		return nil, errors.Errorf("empty public parameters")
	}
	n := network.GetInstance(sp, networkID, channel)
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	networkLocalMembership := n.LocalMembership()
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}

	cs, err := config.GetService(sp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get config service")
	}
	tmsConfig, err := cs.ConfigurationFor(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", networkID, channel, namespace)
	}

	fscIdentity := view.GetIdentityProvider(sp).DefaultIdentity()
	// Prepare roles
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	roles := identity.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	tmsID := token.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(identityDB, sigService, view.GetEndpointService(sp), NewEIDRHDeserializer(), deserializerManager)
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
		sigService,                  // signer service
		view.GetEndpointService(sp), // endpoint service
		storageProvider,
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
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
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

	metricsProvider := metrics.NewTMSProvider(tmsID, metrics.GetProvider(sp))
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(tracing.GetProvider(sp), metricsProvider)
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

func (d *Driver) NewValidator(sp driver.ServiceProvider, tmsID driver.TMSID, params driver.PublicParameters) (driver.Validator, error) {
	logger := logging.DriverLoggerFromPP("token-sdk.driver.zkatdlog", params.Identifier())

	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	deserializer, err := NewDeserializer(pp)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return validator.New(logger, pp, deserializer), nil
	}

	metricsProvider := metrics.NewTMSProvider(tmsID, metrics.GetProvider(sp))
	tracerProvider := tracing2.NewTracerProviderWithBackingProvider(tracing.GetProvider(sp), metricsProvider)
	return observables.NewObservableValidator(
		validator.New(logger, pp, deserializer),
		observables.NewValidator(tracerProvider),
	), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return common.NewPublicParamsManagerFromParams[*crypto.PublicParams](pp)
}

func (d *Driver) NewWalletService(sp driver.ServiceProvider, networkID string, channel string, namespace string, params driver.PublicParameters) (driver.WalletService, error) {
	logger := logging.DriverLogger("token-sdk.driver.zkatdlog", networkID, channel, namespace)

	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	cs, err := config.GetService(sp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get config service")
	}
	tmsConfig, err := cs.ConfigurationFor(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get config for token service for [%s:%s:%s]", networkID, channel, namespace)
	}

	// Prepare roles
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}

	roles := identity.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	tmsID := token.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(identityDB, sigService, nil, NewEIDRHDeserializer(), deserializerManager)
	// public parameters manager
	ppm, err := common.NewPublicParamsManagerFromParams[*crypto.PublicParams](pp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load public parameters")
	}
	identityConfig, err := config2.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create identity config")
	}
	roleFactory := msp.NewRoleFactory(
		tmsID,
		identityConfig, // config
		nil,            // FSC identity
		nil,            // network default identity
		ip,
		sigService, // signer service
		nil,        // endpoint service
		storageProvider,
		deserializerManager,
		true,
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
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	deserializer, err := NewDeserializer(ppm.PublicParams())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate the deserializer")
	}
	// role service
	ws := common.NewWalletService(
		logger,
		ip,
		deserializer,
		zkatdlog.NewWalletFactory(logger, ip, nil, identityConfig, deserializer),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)

	return ws, nil
}

func init() {
	core.Register(crypto.DLogPublicParameters, &Driver{})
}
