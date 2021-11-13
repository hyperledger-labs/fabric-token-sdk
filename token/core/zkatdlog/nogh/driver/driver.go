/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
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

func (d *Driver) NewTokenService(sp view2.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, networkID string, channel string, namespace string) (driver.TokenManagerService, error) {
	n := network.GetInstance(sp, networkID, channel)
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	lm := n.LocalMembership()
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}
	qe := v.TokenVault().QueryEngine()

	nodeIdentity := view2.GetIdentityProvider(sp).DefaultIdentity()
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
		identity.NewProvider(
			sp,
			n,
			map[driver.IdentityUsage]identity.Mapper{
				driver.IssuerRole:  network2.NewMapper(networkID, network2.LongTermIdentity, nodeIdentity, lm),
				driver.AuditorRole: network2.NewMapper(networkID, network2.LongTermIdentity, nodeIdentity, lm),
				driver.OwnerRole:   network2.NewMapper(networkID, network2.AnonymousIdentity, nodeIdentity, lm),
			},
		),
		func(params *crypto.PublicParams) (driver.Deserializer, error) {
			return zkatdlog.NewDeserializer(params)
		},
		crypto.DLogPublicParameters,
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
