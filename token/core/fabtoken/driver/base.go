/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

// base contains the common functionality
type base struct{}

func (d *base) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := fabtoken.NewPublicParamsFromBytes(params, fabtoken.PublicParameters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *base) DefaultValidator(pp driver.PublicParameters) (driver.Validator, error) {
	logger := logging.DriverLoggerFromPP("token-sdk.driver.fabtoken", pp.Identifier())
	deserializer := NewDeserializer()
	return fabtoken.NewValidator(logger, pp.(*fabtoken.PublicParams), deserializer), nil
}

func (d *base) newWalletService(
	tmsConfig driver.Config,
	binder driver2.NetworkBinderService,
	storageProvider identity.StorageProvider,
	qe wallet.TokenVault,
	logger logging.Logger,
	fscIdentity driver.Identity,
	networkDefaultIdentity driver.Identity,
	pp driver.PublicParameters,
	ignoreRemote bool,
) (*wallet.Service, error) {
	tmsID := tmsConfig.ID()

	// Prepare roles
	roles := wallet.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(logger.Named("identity"), identityDB, sigService, binder, NewEIDRHDeserializer())
	identityConfig, err := config2.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create identity config")
	}

	roleFactory := msp.NewRoleFactory(
		logger,
		tmsID,
		identityConfig,         // config
		fscIdentity,            // FSC identity
		networkDefaultIdentity, // network default identity
		ip,
		ip, // signer service
		ip, // endpoint service
		storageProvider,
		deserializerManager,
		ignoreRemote,
	)
	role, err := roleFactory.NewWrappedX509(identity.OwnerRole, ignoreRemote)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles.Register(identity.OwnerRole, role)
	role, err = roleFactory.NewX509(identity.IssuerRole, pp.Issuers()...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(identity.IssuerRole, role)
	role, err = roleFactory.NewX509(identity.AuditorRole, pp.Auditors()...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor role")
	}
	roles.Register(identity.AuditorRole, role)
	role, err = roleFactory.NewX509(identity.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier role")
	}
	roles.Register(identity.CertifierRole, role)

	// Instantiate the token service
	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	ws := wallet.NewService(
		logger,
		ip,
		NewDeserializer(),
		fabtoken.NewWalletFactory(logger, ip, qe),
		roles.ToWalletRegistries(logger, walletDB),
	)

	return ws, nil
}
