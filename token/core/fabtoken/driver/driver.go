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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.driver.fabtoken")

type Driver struct {
}

func (d *Driver) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := fabtoken.NewPublicParamsFromBytes(params, fabtoken.PublicParameters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *Driver) NewTokenService(sp view2.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, networkID string, channel string, namespace string) (driver.TokenManagerService, error) {
	n := network.GetInstance(sp, networkID, channel)
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}
	qe := v.TokenVault().QueryEngine()
	networkLocalMembership := n.LocalMembership()

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
	mspWalletManager.SetRoleIdentityType(driver.OwnerRole, msp.LongTermIdentity)
	mspWalletManager.SetRoleIdentityType(driver.IssuerRole, msp.LongTermIdentity)
	mspWalletManager.SetRoleIdentityType(driver.AuditorRole, msp.LongTermIdentity)
	mspWalletManager.SetRoleIdentityType(driver.CertifierRole, msp.LongTermIdentity)
	if err := mspWalletManager.Load(); err != nil {
		return nil, errors.WithMessage(err, "failed to load wallets")
	}
	wallets, err := mspWalletManager.Wallets()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get wallets")
	}

	return fabtoken.NewService(
		sp,
		channel,
		namespace,
		fabtoken.NewPublicParamsManager(&fabtoken.VaultPublicParamsLoader{
			TokenVault:          qe,
			PublicParamsFetcher: publicParamsFetcher,
			PPLabel:             fabtoken.PublicParameters,
		}),
		&fabtoken.VaultTokenLoader{TokenVault: qe},
		qe,
		identity.NewProvider(sp, fabtoken.NewEnrollmentIDDeserializer(), wallets),
		fabtoken.NewDeserializer(),
		tmsConfig,
	), nil
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return fabtoken.NewValidator(pp, fabtoken.NewDeserializer()), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return fabtoken.NewPublicParamsManagerFromParams(pp), nil
}

func init() {
	core.Register(fabtoken.PublicParameters, &Driver{})
}
