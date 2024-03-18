/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.driver.zkatdlog")

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

	cs := view.GetConfigService(sp)
	tmsConfig, err := config.NewTokenSDK(cs).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	fscIdentity := view.GetIdentityProvider(sp).DefaultIdentity()
	// Prepare roles
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	roles := identity.NewRoles()
	deserializerManager := deserializer.NewMultiplexDeserializer()
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
	roleFactory := msp.NewRoleFactory(
		tmsID,
		config2.NewIdentityConfig(cs, tmsConfig), // config
		fscIdentity,                              // FSC identity
		networkLocalMembership.DefaultIdentity(), // network default identity
		sigService,                               // signer service
		view.GetEndpointService(sp),              // endpoint service
		storageProvider,
		deserializerManager,
		false,
	)
	role, err := roleFactory.NewIdemix(driver.OwnerRole, tmsConfig.TMS().GetWalletDefaultCacheSize(), math.BLS12_381_BBS)
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
	ppm := ppm.NewPublicParamsManager(crypto.DLogPublicParameters, qe)
	ppm.AddCallback(func(pp driver.PublicParameters) error {
		return roles.Reload(pp)
	})
	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}

	ip := identity.NewProvider(sigService, view.GetEndpointService(sp), NewEnrollmentIDDeserializer(), deserializerManager)
	deserializerProvider := NewDeserializerProvider(ppm).Deserialize
	ws := common.NewWalletService(
		logger,
		ip,
		deserializerProvider,
		zkatdlog.NewWalletFactory(ip, qe, tmsConfig, deserializerProvider),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)
	service, err := zkatdlog.NewTokenService(
		ws,
		ppm,
		&zkatdlog.VaultTokenLoader{TokenVault: qe},
		zkatdlog.NewVaultTokenCommitmentLoader(qe, 3, 3*time.Second),
		ip,
		deserializerProvider,
		tmsConfig,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}
	if err := ppm.SetPublicParameters(publicParams); err != nil {
		return nil, errors.WithMessage(err, "failed to fetch public parameters")
	}

	return service, err
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	deserializer, err := NewDeserializer(pp)
	if err != nil {
		return nil, err
	}
	return validator.New(pp, deserializer), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return ppm.NewFromParams(pp)
}

func (d *Driver) NewWalletService(sp driver.ServiceProvider, networkID string, channel string, namespace string, params driver.PublicParameters) (driver.WalletService, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	cs := view.GetConfigService(sp)
	tmsConfig, err := config.NewTokenSDK(cs).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	// Prepare roles
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}

	roles := identity.NewRoles()
	deserializerManager := deserializer.NewMultiplexDeserializer()
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
	roleFactory := msp.NewRoleFactory(
		tmsID,
		config2.NewIdentityConfig(cs, tmsConfig), // config
		nil,                                      // FSC identity
		nil,                                      // network default identity
		sigService,                               // signer service
		nil,                                      // endpoint service
		storageProvider,
		deserializerManager,
		true,
	)
	role, err := roleFactory.NewIdemix(driver.OwnerRole, tmsConfig.TMS().GetWalletDefaultCacheSize(), math.BN254)
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

	// public parameters manager
	publicParamsManager, err := ppm.NewFromParams(pp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load public parameters")
	}

	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	ip := identity.NewProvider(sigService, nil, NewEnrollmentIDDeserializer(), deserializerManager)
	deserializerProvider := NewDeserializerProvider(publicParamsManager).Deserialize
	// role service
	ws := common.NewWalletService(
		logger,
		ip,
		deserializerProvider,
		zkatdlog.NewWalletFactory(ip, nil, tmsConfig, deserializerProvider),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)

	if err := roles.Reload(pp); err != nil {
		return nil, errors.WithMessage(err, "failed to load roles")
	}

	return ws, nil
}

func init() {
	core.Register(crypto.DLogPublicParameters, &Driver{})
}
