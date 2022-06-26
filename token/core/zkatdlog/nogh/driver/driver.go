/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
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

func (d *Driver) NewTokenService(sp view2.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, networkID string, channel string, namespace string) (driver.TokenManagerService, error) {
	n := network.GetInstance(sp, networkID, channel)
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	networkLocalMembership := n.LocalMembership()
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}
	qe := v.TokenVault().QueryEngine()

	tmsConfig, err := config.NewTokenSDK(view2.GetConfigService(sp)).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	mspWalletManager := msp.NewWalletManager(
		sp,        // service provider
		networkID, // network ID
		tmsConfig, // config manager
		view2.GetIdentityProvider(sp).DefaultIdentity(), // FSC identity
		networkLocalMembership.DefaultIdentity(),        // network default identity
		msp.NewSigService(view2.GetSigService(sp)),      // signer service
		view2.GetEndpointService(sp),                    // endpoint service
	)
	mspWalletManager.SetRoleIdentityType(driver.OwnerRole, msp.AnonymousIdentity)
	mspWalletManager.SetRoleIdentityType(driver.IssuerRole, msp.LongTermIdentity)
	mspWalletManager.SetRoleIdentityType(driver.AuditorRole, msp.LongTermIdentity)
	if err := mspWalletManager.Load(); err != nil {
		return nil, errors.WithMessage(err, "failed to load wallets")
	}
	mappers, err := mspWalletManager.Mappers()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get wallet mappers")
	}

	desProvider := zkatdlog.NewDeserializerProvider()
	service, err := zkatdlog.NewTokenService(
		channel,
		namespace,
		sp,
		ppm.New(&zkatdlog.VaultPublicParamsLoader{
			TokenVault:          qe,
			PublicParamsFetcher: publicParamsFetcher,
			PPLabel:             crypto.DLogPublicParameters,
		}),
		&zkatdlog.VaultTokenLoader{TokenVault: v.TokenVault().QueryEngine()},
		&zkatdlog.VaultTokenCommitmentLoader{TokenVault: v.TokenVault().QueryEngine()},
		v.TokenVault().QueryEngine(),
		identity.NewProvider(sp, zkatdlog.NewEnrollmentIDDeserializer(), mappers),
		desProvider.Deserialize,
		crypto.DLogPublicParameters,
		tmsConfig,
	)
	if err != nil {
		return nil, err
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
	return ppm.NewFromParams(pp), nil
}

func init() {
	core.Register(crypto.DLogPublicParameters, &Driver{})
}
