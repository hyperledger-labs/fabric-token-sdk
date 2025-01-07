/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
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
	binder common2.NetworkBinderService,
	storageProvider identity.StorageProvider,
	qe common.TokenVault,
	logger logging.Logger,
	fscIdentity driver.Identity,
	networkDefaultIdentity driver.Identity,
	pp driver.PublicParameters,
	ignoreRemote bool,
) (*common.WalletService, error) {
	tmsID := tmsConfig.ID()

	// Prepare roles
	roles := identity.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(identityDB, sigService, binder, NewEIDRHDeserializer())
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
	role, err := roleFactory.NewWrappedX509(driver.OwnerRole, ignoreRemote)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles.Register(driver.OwnerRole, role)
	role, err = roleFactory.NewX509(driver.IssuerRole, pp.Issuers()...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(driver.IssuerRole, role)
	role, err = roleFactory.NewX509(driver.AuditorRole, pp.Auditors()...)
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
	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	ws := common.NewWalletService(
		logger,
		ip,
		NewDeserializer(),
		fabtoken.NewWalletFactory(logger, ip, qe),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)

	return ws, nil
}
