/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"time"

	math "github.com/IBM/mathlib"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	weaver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/weaver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/state/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver/state/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

type Driver struct{}

func (d *Driver) NewStateQueryExecutor(sp driver.ServiceProvider, url string) (driver.StateQueryExecutor, error) {
	return fabric3.NewStateQueryExecutor(weaver2.GetProvider(sp), url, fabric2.GetDefaultFNS(sp))
}

func (d *Driver) NewStateVerifier(sp driver.ServiceProvider, url string) (driver.StateVerifier, error) {
	return fabric3.NewStateVerifier(
		weaver2.GetProvider(sp),
		pledge.Vault(sp),
		func(id string) *fabric2.NetworkService {
			return fabric2.GetFabricNetworkService(sp, id)
		},
		url,
		fabric2.GetDefaultFNS(sp),
	)
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

	fscIdentity := view.GetIdentityProvider(sp).DefaultIdentity()
	// Prepare wallets
	wallets := identity.NewWallets()
	dsManager, err := common.GetDeserializerManager(sp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get deserializer manager")
	}
	mspWalletFactory := msp.NewWalletFactory(
		sp,                                       // service provider
		networkID,                                // network ID
		tmsConfig,                                // config manager
		fscIdentity,                              // FSC identity
		networkLocalMembership.DefaultIdentity(), // network default identity
		msp.NewSigService(view.GetSigService(sp)), // signer service
		view.GetEndpointService(sp),               // endpoint service
		kvs.GetService(sp),
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
	qe := v.TokenVault().QueryEngine()
	ppm := ppm.NewPublicParamsManager(
		crypto.DLogPublicParameters,
		v.TokenVault().QueryEngine(),
		zkatdlog.NewPublicParamsLoader(publicParamsFetcher, crypto.DLogPublicParameters),
	)
	ppm.AddCallback(func(pp driver.PublicParameters) error {
		return wallets.Reload(pp)
	})
	// wallet service
	ws := zkatdlog.NewWalletService(
		tmsID,
		msp.NewSigService(view.GetSigService(sp)),
		identity.NewProvider(view.GetSigService(sp), view.GetEndpointService(sp), fscIdentity, zkatdlog.NewEnrollmentIDDeserializer(), wallets),
		qe,
		ppm,
		zkatdlog.NewDeserializerProvider().Deserialize,
		tmsConfig,
		kvs.GetService(sp),
	)
	service, err := zkatdlog.NewTokenService(
		ws,
		ppm,
		&zkatdlog.VaultTokenLoader{TokenVault: qe},
		zkatdlog.NewVaultTokenCommitmentLoader(qe, 3, 3*time.Second),
		identity.NewProvider(view.GetSigService(sp), view.GetEndpointService(sp), fscIdentity, zkatdlog.NewEnrollmentIDDeserializer(), wallets),
		zkatdlog.NewDeserializerProvider().Deserialize,
		tmsConfig,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}
	if err := ppm.Load(); err != nil {
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

func (d *Driver) NewWalletService(sp view.ServiceProvider, networkID string, channel string, namespace string, params driver.PublicParameters) (driver.WalletService, error) {
	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	tmsConfig, err := config.NewTokenSDK(view.GetConfigService(sp)).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	// Prepare wallets
	wallets := identity.NewWallets()
	dsManager, err := common.GetDeserializerManager(sp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get deserializer manager")
	}
	mspWalletFactory := msp.NewWalletFactory(
		sp,        // service provider
		networkID, // network ID
		tmsConfig, // config manager
		nil,       // FSC identity
		nil,       // network default identity
		msp.NewSigService(view.GetSigService(sp)), // signer service
		nil, // endpoint service
		kvs.GetService(sp),
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
	// wallet service
	ws := zkatdlog.NewWalletService(
		tmsID,
		msp.NewSigService(view.GetSigService(sp)),
		identity.NewProvider(view.GetSigService(sp), nil, nil, zkatdlog.NewEnrollmentIDDeserializer(), wallets),
		nil,
		publicParamsManager,
		zkatdlog.NewDeserializerProvider().Deserialize,
		tmsConfig,
		kvs.GetService(sp),
	)

	if err := wallets.Reload(pp); err != nil {
		return nil, errors.WithMessage(err, "failed to load wallets")
	}

	return ws, nil
}

func init() {
	d := &Driver{}
	core.Register(crypto.DLogPublicParameters, d)
	fabric.RegisterStateDriver(crypto.DLogPublicParameters, d)
}
