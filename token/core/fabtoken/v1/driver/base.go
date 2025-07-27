/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	core2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/role"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/logging"
	"github.com/pkg/errors"
)

// base contains the common functionality
type base struct{}

func (d *base) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := core2.NewPublicParamsFromBytes(params, core2.FabTokenDriverName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *base) DefaultValidator(pp driver.PublicParameters) (driver.Validator, error) {
	logger := logging.DriverLoggerFromPP("token-sdk.driver.fabtoken", string(core.DriverIdentifierFromPP(pp)))
	deserializer := NewDeserializer()
	return validator.NewValidator(logger, pp.(*core2.PublicParams), deserializer, nil, nil, nil), nil
}

func (d *base) newWalletService(
	tmsConfig core.Config,
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
	identityDB, err := storageProvider.IdentityStore(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	baseKeyStore, err := storageProvider.Keystore()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open keystore for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	identityProvider := identity.NewProvider(logger.Named("identity"), identityDB, sigService, binder, NewEIDRHDeserializer())
	identityConfig, err := config.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create identity config")
	}

	// Prepare roles
	keyStore := x509.NewKeyStore(baseKeyStore)
	roleFactory := role.NewFactory(logger, tmsID, identityConfig, fscIdentity, networkDefaultIdentity, identityProvider, identityProvider, identityProvider, storageProvider, deserializerManager)
	role, err := roleFactory.NewRole(identity.OwnerRole, false, nil, x509.NewKeyManagerProvider(identityConfig, identityProvider, keyStore, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles := wallet.NewRoles()
	roles.Register(identity.OwnerRole, role)
	role, err = roleFactory.NewRole(identity.IssuerRole, false, pp.Issuers(), x509.NewKeyManagerProvider(identityConfig, identityProvider, keyStore, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(identity.IssuerRole, role)
	role, err = roleFactory.NewRole(identity.AuditorRole, false, pp.Auditors(), x509.NewKeyManagerProvider(identityConfig, identityProvider, keyStore, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor role")
	}
	roles.Register(identity.AuditorRole, role)
	role, err = roleFactory.NewRole(identity.CertifierRole, false, nil, x509.NewKeyManagerProvider(identityConfig, identityProvider, keyStore, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier role")
	}
	roles.Register(identity.CertifierRole, role)

	// Instantiate the wallet service
	walletDB, err := storageProvider.WalletStore(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	deserializer := NewDeserializer()
	ws := wallet.NewService(
		logger,
		identityProvider,
		deserializer,
		wallet.NewFactory(logger, identityProvider, qe, identityConfig, deserializer, &disabled.Provider{}),
		roles.ToWalletRegistries(logger, walletDB),
	)

	return ws, nil
}
