/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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

func (d *Driver) NewTokenService(sp view.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, networkID string, channel string, namespace string) (driver.TokenManagerService, error) {
	n := network.GetInstance(sp, networkID, channel)
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	networkLocalMembership := n.LocalMembership()
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}

	tmsConfig, err := config.NewTokenSDK(view.GetConfigService(sp)).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	// Prepare wallets
	wallets := identity.NewWallets()
	mspWalletFactory := msp.NewWalletFactory(
		sp,        // service provider
		networkID, // network ID
		tmsConfig, // config manager
		view.GetIdentityProvider(sp).DefaultIdentity(), // FSC identity
		networkLocalMembership.DefaultIdentity(),       // network default identity
		msp.NewSigService(view.GetSigService(sp)),      // signer service
		view.GetEndpointService(sp),                    // endpoint service
	)
	wallet, err := mspWalletFactory.NewIdemixWallet(driver.OwnerRole, tmsConfig.TMS().GetWalletDefaultCacheSize())
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
	qe := v.TokenVault().QueryEngine()
	service, err := zkatdlog.NewTokenService(
		sp,
		tmsID,
		ppm.NewPublicParamsManager(
			crypto.DLogPublicParameters,
			v.TokenVault().QueryEngine(),
			zkatdlog.NewPublicParamsLoader(publicParamsFetcher, crypto.DLogPublicParameters)),
		&zkatdlog.VaultTokenLoader{TokenVault: qe},
		zkatdlog.NewVaultTokenCommitmentLoader(qe, 3, 3*time.Second),
		qe,
		identity.NewProvider(sp, zkatdlog.NewEnrollmentIDDeserializer(), wallets),
		zkatdlog.NewDeserializerProvider().Deserialize,
		crypto.DLogPublicParameters,
		tmsConfig,
		kvs.GetService(sp),
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}
	if err := service.LoadPublicParams(); err != nil {
		return nil, errors.WithMessage(err, "failed to fetch public parameters")
	}

	return service, err
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	deserializer, err := zkatdlog.NewDeserializer(pp)
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

func init() {
	core.Register(crypto.DLogPublicParameters, &Driver{})
}
