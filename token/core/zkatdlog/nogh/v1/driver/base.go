/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/role"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/wallet"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

//go:generate counterfeiter -o mock/config.go -fake-name Config . Config
type Config = core.Config

type Base struct{}

func (d *Base) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := v1.NewPublicParamsFromBytes(params, v1.DLogNoGHDriverName, v1.ProtocolV1)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *Base) DefaultValidator(pp driver.PublicParameters) (driver.Validator, error) {
	deserializer, err := NewDeserializer(pp.(*v1.PublicParams))
	if err != nil {
		return nil, errors.Errorf("failed to create token service deserializer: %v", err)
	}
	logger := logging.DriverLoggerFromPP("token-sdk.driver.zkatdlog", string(pp.TokenDriverName()))
	return validator.New(
		logger,
		pp.(*v1.PublicParams),
		deserializer,
		nil,
		nil,
		nil,
	), nil
}

func (d *Base) NewWalletService(
	tmsConfig core.Config,
	binder idriver.NetworkBinderService,
	storageProvider identity.StorageProvider,
	qe driver.QueryEngine,
	logger logging.Logger,
	fscIdentity view.Identity,
	networkDefaultIdentity view.Identity,
	publicParams driver.PublicParameters,
	ignoreRemote bool,
	metricsProvider metrics.Provider,
) (*wallet.Service, error) {
	pp := publicParams.(*v1.PublicParams)
	roles := role.NewRoles()
	deserializerManager := deserializer.NewTypedSignerDeserializerMultiplex()
	tmsID := tmsConfig.ID()
	identityDB, err := storageProvider.IdentityStore(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	baseKeyStore, err := storageProvider.Keystore(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open keystore for tms [%s]", tmsID)
	}
	identityProvider := identity.NewProvider(logger.Named("identity"), identityDB, deserializerManager, binder, NewEIDRHDeserializer())
	identityConfig, err := config.NewIdentityConfig(tmsConfig)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create identity config")
	}

	// Prepare roles
	roleFactory := membership.NewRoleFactory(
		logger,
		tmsID,
		identityConfig,
		fscIdentity,
		networkDefaultIdentity,
		identityProvider,
		storageProvider,
		deserializerManager,
	)
	// owner role
	// we have one key manager for fabtoken and one for each idemix issuer public key
	kmps := make([]membership.KeyManagerProvider, 0, len(pp.IdemixIssuerPublicKeys)+1)
	for _, key := range pp.IdemixIssuerPublicKeys {
		keyStore, err := msp2.NewKeyStore(key.Curve, baseKeyStore)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate bccsp key store")
		}
		kmp := idemix.NewKeyManagerProvider(key.PublicKey, key.Curve, keyStore, identityConfig, identityConfig.DefaultCacheSize(), ignoreRemote, metricsProvider)
		kmps = append(kmps, kmp)
	}
	keyStore := x509.NewKeyStore(baseKeyStore)
	kmps = append(kmps, x509.NewKeyManagerProvider(identityConfig, keyStore, ignoreRemote))

	newRole, err := roleFactory.NewRole(identity.OwnerRole, true, nil, kmps...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create owner role")
	}
	roles.Register(identity.OwnerRole, newRole)
	newRole, err = roleFactory.NewRole(identity.IssuerRole, false, pp.Issuers(), x509.NewKeyManagerProvider(identityConfig, keyStore, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create issuer role")
	}
	roles.Register(identity.IssuerRole, newRole)
	newRole, err = roleFactory.NewRole(identity.AuditorRole, false, pp.Auditors(), x509.NewKeyManagerProvider(identityConfig, keyStore, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create auditor role")
	}
	roles.Register(identity.AuditorRole, newRole)
	newRole, err = roleFactory.NewRole(identity.CertifierRole, false, nil, x509.NewKeyManagerProvider(identityConfig, keyStore, ignoreRemote))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create certifier role")
	}
	roles.Register(identity.CertifierRole, newRole)

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
		wallet.Convert(roles.Registries(logger, walletDB, role.NewDefaultFactory(logger, identityProvider, qe, identityConfig, deserializer, metricsProvider))),
	), nil
}
