/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type base struct{}

func (d *base) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := crypto.NewPublicParamsFromBytes(params, crypto.DLogPublicParameters)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *base) DefaultValidator(pp driver.PublicParameters) (driver.Validator, error) {
	deserializer, err := NewDeserializer(pp.(*crypto.PublicParams))
	if err != nil {
		return nil, errors.Errorf("failed to create token service deserializer: %v", err)
	}
	logger := logging.DriverLoggerFromPP("token-sdk.driver.zkatdlog", pp.Identifier())
	return validator.New(logger, pp.(*crypto.PublicParams), deserializer), nil
}

func (d *base) newWalletService(
	tmsConfig driver.Config,
	binder idriver.NetworkBinderService,
	storageProvider identity.StorageProvider,
	qe driver.QueryEngine,
	logger logging.Logger,
	fscIdentity view3.Identity,
	networkDefaultIdentity view3.Identity,
	publicParams driver.PublicParameters,
	ignoreRemote bool,
) (*wallet.Service, error) {
	pp := publicParams.(*crypto.PublicParams)
	// Prepare roles
	roles := wallet.NewRoles()
	deserializerManager := sig.NewMultiplexDeserializer()
	tmsID := tmsConfig.ID()
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
	// owner role
	// we have one key manager for fabtoken and one for each idemix issuer public key
	kmps := make([]common2.KeyManagerProvider, 0, len(pp.IdemixIssuerPublicKeys)+1)
	for _, key := range pp.IdemixIssuerPublicKeys {
		backend, err := storageProvider.NewKeystore()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get new keystore backend")
		}
		keyStore, err := msp2.NewKeyStore(key.Curve, backend)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate bccsp key store")
		}
		kmp := idemix2.NewKeyManagerProvider(
			key.PublicKey,
			key.Curve,
			msp.RoleToMSPID[driver.OwnerRole],
			keyStore,
			sigService,
			identityConfig,
			identityConfig.DefaultCacheSize(),
			ignoreRemote,
		)
		kmps = append(kmps, kmp)
	}
	kmps = append(kmps, x509.NewKeyManagerProvider(identityConfig, msp.RoleToMSPID[driver.OwnerRole], ip, ignoreRemote))

	role, err := roleFactory.NewIdemix(
		identity.OwnerRole,
		identityConfig.DefaultCacheSize(),
		nil,
		kmps...,
	)
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
	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}

	deserializer, err := NewDeserializer(pp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate the deserializer")
	}
	return wallet.NewService(
		logger,
		ip,
		deserializer,
		zkatdlog.NewWalletFactory(logger, ip, qe, identityConfig, deserializer),
		roles.ToWalletRegistries(logger, walletDB),
	), nil
}
