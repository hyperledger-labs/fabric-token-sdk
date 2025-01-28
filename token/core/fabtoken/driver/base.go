/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/role"
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
	qe driver.QueryEngine,
	logger logging.Logger,
	fscIdentity driver.Identity,
	networkDefaultIdentity driver.Identity,
	pp driver.PublicParameters,
	ignoreRemote bool,
) (*wallet.Service, error) {
	tmsID := tmsConfig.ID()

	deserializerManager := sig.NewMultiplexDeserializer()
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(logger.Named("identity"), identityDB, sigService, binder, NewEIDRHDeserializer())
	identityConfig, err := config.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create identity config")
	}

	// Prepare roles
	roleFactory := role.NewFactory(logger, tmsID, identityConfig, fscIdentity, networkDefaultIdentity, ip, ip, ip, storageProvider, deserializerManager)
	role, err := roleFactory.NewRole(identity.OwnerRole, false, nil, x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.OwnerRole], ip, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles := wallet.NewRoles()
	roles.Register(identity.OwnerRole, role)
	role, err = roleFactory.NewRole(identity.IssuerRole, false, pp.Issuers(), x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.IssuerRole], ip, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(identity.IssuerRole, role)
	role, err = roleFactory.NewRole(identity.AuditorRole, false, pp.Auditors(), x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.AuditorRole], ip, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor role")
	}
	roles.Register(identity.AuditorRole, role)
	role, err = roleFactory.NewRole(identity.CertifierRole, false, nil, x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[identity.CertifierRole], ip, ignoreRemote))
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
	deserializer := NewDeserializer()
	ws := wallet.NewService(
		logger,
		ip,
		deserializer,
		wallet.NewFactory(logger, ip, qe, identityConfig, deserializer),
		roles.ToWalletRegistries(logger, walletDB),
	)

	return ws, nil
}
