/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import (
	"fmt"
	"reflect"
	"sync"

	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"

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

type DeserializerManager interface {
	AddDeserializer(deserializer sig2.Deserializer)
}

var once sync.Once

type Driver struct {
}

func (d *Driver) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := crypto.NewPublicParamsFromBytes(params, crypto.DLogPublicParameters)
	if err != nil {
		return nil, err
	}
	return pp, nil
}

func (d *Driver) NewTokenService(sp view2.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, network string, channel driver.Channel, namespace string) (driver.TokenManagerService, error) {
	nodeIdentity := view2.GetIdentityProvider(sp).DefaultIdentity()
	service, err := zkatdlog.NewTokenService(
		channel,
		namespace,
		sp,
		publicParamsFetcher,
		&zkatdlog.VaultTokenCommitmentLoader{TokenVault: vault.NewVault(sp, channel, namespace).QueryEngine()},
		vault.NewVault(sp, channel, namespace).QueryEngine(),
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
	once.Do(func() {
		// Register deserializers
		dm, err := sp.GetService(reflect.TypeOf((*DeserializerManager)(nil)))
		if err != nil {
			panic(fmt.Sprintf("failed looking up deserializer manager [%s]", err))
		}
		dm.(DeserializerManager).AddDeserializer(&Deserializer{Service: service})
	})

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
