/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
)

type Driver struct {
}

func (d *Driver) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := crypto.NewPublicParamsFromBytes(params, crypto.DLogPublicParameters)
	if err != nil {
		return nil, err
	}
	return pp, nil
}

func (d *Driver) NewTokenService(sp view2.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, network string, channel string, namespace string) (driver.TokenManagerService, error) {
	fns := fabric2.GetFabricNetworkService(sp, network)
	if fns == nil {
		return nil, errors.Errorf("fabric network [%s] does not exists", network)
	}
	ch, err := fns.Channel(channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric channel [%s:%s] does not exists", network, channel)
	}

	nodeIdentity := view2.GetIdentityProvider(sp).DefaultIdentity()
	service, err := zkatdlog.NewTokenService(
		ch,
		namespace,
		sp,
		publicParamsFetcher,
		&zkatdlog.VaultTokenCommitmentLoader{TokenVault: vault.NewVault(sp, ch, namespace).QueryEngine()},
		vault.NewVault(sp, ch, namespace).QueryEngine(),
		identity.NewProvider(
			sp,
			map[driver.IdentityUsage]identity.Mapper{
				driver.IssuerRole:  fabric.NewMapper(network, fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, network).LocalMembership()),
				driver.AuditorRole: fabric.NewMapper(network, fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, network).LocalMembership()),
				driver.OwnerRole:   fabric.NewMapper(network, fabric.IdemixMSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, network).LocalMembership()),
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
	pp := params.(*crypto.PublicParams)
	deserializer, err := zkatdlog.NewDeserializer(pp)
	if err != nil {
		return nil, err
	}
	return validator.New(pp, deserializer), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	return ppm.New(params.(*crypto.PublicParams)), nil
}

func init() {
	core.Register(crypto.DLogPublicParameters, &Driver{})
}
