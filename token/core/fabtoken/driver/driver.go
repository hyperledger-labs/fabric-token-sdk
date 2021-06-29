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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
)

var once sync.Once

type DeserializerManager interface {
	AddDeserializer(deserializer sig2.Deserializer)
}

type Driver struct {
}

func (d *Driver) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := fabtoken.NewPublicParamsFromBytes(params)
	if err != nil {
		return nil, err
	}
	return pp, nil
}

func (d *Driver) NewTokenService(sp view2.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, network string, channel driver.Channel, namespace string) (driver.TokenManagerService, error) {
	once.Do(func() {
		// Register deserializers
		dm, err := sp.GetService(reflect.TypeOf((*DeserializerManager)(nil)))
		if err != nil {
			panic(fmt.Sprintf("failed looking up deserializer manager [%s]", err))
		}
		dm.(DeserializerManager).AddDeserializer(fabtoken.NewRawOwnerIdentityDeserializer())
	})

	qe := vault.NewVault(sp, channel, namespace).QueryEngine()
	nodeIdentity := view2.GetIdentityProvider(sp).DefaultIdentity()
	return fabtoken.NewService(
		sp,
		channel,
		namespace,
		publicParamsFetcher,
		&fabtoken.VaultPublicParamsLoader{
			TokenVault:          qe,
			PublicParamsFetcher: publicParamsFetcher,
		},
		qe,
		identity.NewProvider(
			sp,
			map[driver.IdentityUsage]identity.Mapper{
				driver.IssuerRole:  fabric.NewMapper(fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, network).LocalMembership()),
				driver.AuditorRole: fabric.NewMapper(fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, network).LocalMembership()),
				driver.OwnerRole:   fabric.NewMapper(fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, network).LocalMembership()),
			},
		),
	), nil
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	return fabtoken.NewValidator(params.(*fabtoken.PublicParams)), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	return fabtoken.NewPublicParamsManager(params.(*fabtoken.PublicParams)), nil
}

func init() {
	core.Register(fabtoken.PublicParameters, &Driver{})
}
