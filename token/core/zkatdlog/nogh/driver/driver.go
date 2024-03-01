/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"time"

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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
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
	// Prepare wallets
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	wallets := identity.NewWallets()
	dsManager, err := common.GetDeserializerManager(sp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get deserializer manager")
	}
	mspWalletFactory := msp.NewWalletFactory(
		token.TMSID{
			Network:   networkID,
			Channel:   channel,
			Namespace: namespace,
		},
		config.NewIdentityConfig(cs, tmsConfig),   // config
		fscIdentity,                               // FSC identity
		networkLocalMembership.DefaultIdentity(),  // network default identity
		msp.NewSigService(view.GetSigService(sp)), // signer service
		view.GetEndpointService(sp),               // endpoint service
		storageProvider,
		dsManager,
		false,
	)
	wallet, err := mspWalletFactory.NewIdemixWallet(driver.OwnerRole, tmsConfig.TMS().GetWalletDefaultCacheSize(), math.BLS12_381_BBS)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner wallet")
	}
	wallets.Put(driver.OwnerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.IssuerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer wallet")
	}
	wallets.Put(driver.IssuerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.AuditorRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor wallet")
	}
	wallets.Put(driver.AuditorRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier wallet")
	}
	wallets.Put(driver.CertifierRole, wallet)

	// Instantiate the token service
	tmsID := token.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	qe := v.QueryEngine()
	ppm := ppm.NewPublicParamsManager(crypto.DLogPublicParameters, qe)
	ppm.AddCallback(func(pp driver.PublicParameters) error {
		return wallets.Reload(pp)
	})
	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	ip := identity.NewProvider(
		view.GetSigService(sp),
		view.GetEndpointService(sp),
		fscIdentity,
		NewEnrollmentIDDeserializer(),
		wallets,
	)
	ws := zkatdlog.NewWalletService(
		msp.NewSigService(view.GetSigService(sp)),
		ip,
		qe,
		ppm,
		NewDeserializerProvider().Deserialize,
		tmsConfig,
		identity.NewWalletsRegistry(ip, driver.OwnerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.IssuerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.AuditorRole, walletDB),
	)
	service, err := zkatdlog.NewTokenService(
		ws,
		ppm,
		&zkatdlog.VaultTokenLoader{TokenVault: qe},
		zkatdlog.NewVaultTokenCommitmentLoader(qe, 3, 3*time.Second),
		ip,
		NewDeserializerProvider().Deserialize,
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

	// Prepare wallets
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}

	wallets := identity.NewWallets()
	dsManager, err := common.GetDeserializerManager(sp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get deserializer manager")
	}
	mspWalletFactory := msp.NewWalletFactory(
		token.TMSID{
			Network:   networkID,
			Channel:   channel,
			Namespace: namespace,
		},
		config.NewIdentityConfig(cs, tmsConfig), // config
		nil,                                     // FSC identity
		nil,                                     // network default identity
		msp.NewSigService(view.GetSigService(sp)), // signer service
		nil, // endpoint service
		storageProvider,
		dsManager,
		true,
	)
	wallet, err := mspWalletFactory.NewIdemixWallet(driver.OwnerRole, tmsConfig.TMS().GetWalletDefaultCacheSize(), math.BN254)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner wallet")
	}
	wallets.Put(driver.OwnerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.IssuerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer wallet")
	}
	wallets.Put(driver.IssuerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.AuditorRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor wallet")
	}
	wallets.Put(driver.AuditorRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier wallet")
	}
	wallets.Put(driver.CertifierRole, wallet)

	// Instantiate the token service
	tmsID := token.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	// public parameters manager
	publicParamsManager, err := ppm.NewFromParams(pp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load public parameters")
	}

	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	ip := identity.NewProvider(
		view.GetSigService(sp),
		nil,
		nil,
		NewEnrollmentIDDeserializer(),
		wallets,
	)
	// wallet service
	ws := zkatdlog.NewWalletService(
		msp.NewSigService(view.GetSigService(sp)),
		ip,
		nil,
		publicParamsManager,
		NewDeserializerProvider().Deserialize,
		tmsConfig,
		identity.NewWalletsRegistry(ip, driver.OwnerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.IssuerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.AuditorRole, walletDB),
	)

	if err := wallets.Reload(pp); err != nil {
		return nil, errors.WithMessage(err, "failed to load wallets")
	}

	return ws, nil
}

func init() {
	core.Register(crypto.DLogPublicParameters, &Driver{})
}
