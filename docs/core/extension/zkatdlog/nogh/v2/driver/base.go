/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	v2 "github.com/hyperledger-labs/fabric-token-sdk/docs/core/extension/zkatdlog/nogh/v2/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/role"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type base struct{}

func (d *base) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := v2.NewPublicParamsFromBytes(params, v2.DLogIdentifier, v2.ProtocolV2)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *base) DefaultValidator(pp driver.PublicParameters) (driver.Validator, error) {
	deserializer, err := NewDeserializer(pp.(*v2.PublicParams))
	if err != nil {
		return nil, errors.Errorf("failed to create token service deserializer: %v", err)
	}
	logger := logging.DriverLoggerFromPP("token-sdk.driver.zkatdlog", string(pp.TokenDriverName()))
	return validator.New(
		logger,
		pp.(*v2.PublicParams),
		deserializer,
		nil,
		nil,
		nil,
	), nil
}

func (d *base) newWalletService(
	tmsConfig core.Config,
	binder idriver.NetworkBinderService,
	storageProvider identity.StorageProvider,
	qe driver.QueryEngine,
	logger logging.Logger,
	fscIdentity view.Identity,
	networkDefaultIdentity view.Identity,
	publicParams driver.PublicParameters,
	ignoreRemote bool,
) (*wallet.Service, error) {
	pp := publicParams.(*v2.PublicParams)
	roles := wallet.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	tmsID := tmsConfig.ID()
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
	roleFactory := role.NewFactory(logger, tmsID, identityConfig, fscIdentity, networkDefaultIdentity, identityProvider, identityProvider, identityProvider, storageProvider, deserializerManager)
	// owner role
	// we have one key manager for fabtoken and one for each idemix issuer public key
	kmps := make([]membership.KeyManagerProvider, 0, len(pp.IdemixIssuerPublicKeys)+1)
	for _, key := range pp.IdemixIssuerPublicKeys {
		backend, err := storageProvider.Keystore()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get new keystore backend")
		}
		keyStore, err := msp2.NewKeyStore(key.Curve, backend)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate bccsp key store")
		}
		kmp := idemix.NewKeyManagerProvider(key.PublicKey, key.Curve, keyStore, sigService, identityConfig, identityConfig.DefaultCacheSize(), ignoreRemote)
		kmps = append(kmps, kmp)
	}
	keyStore := x509.NewKeyStore(baseKeyStore)
	kmps = append(kmps, x509.NewKeyManagerProvider(identityConfig, identityProvider, keyStore, ignoreRemote))

	role, err := roleFactory.NewRole(identity.OwnerRole, true, nil, kmps...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
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

	// wallet service
	walletDB, err := storageProvider.WalletStore(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	deserializer, err := NewDeserializer(pp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate the deserializer")
	}
	return wallet.NewService(
		logger,
		identityProvider,
		deserializer,
		wallet.NewFactory(logger, identityProvider, qe, identityConfig, deserializer),
		roles.ToWalletRegistries(logger, walletDB),
	), nil
}
