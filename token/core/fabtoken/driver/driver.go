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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type Driver struct {
}

func (d *Driver) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := fabtoken.NewPublicParamsFromBytes(params, fabtoken.PublicParameters)
	if err != nil {
		return nil, err
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
	nodeIdentity := view2.GetIdentityProvider(sp).DefaultIdentity()
	return fabtoken.NewService(
		sp,
		namespace,
		fabtoken.NewPublicParamsManager(&fabtoken.VaultPublicParamsLoader{
			TokenVault:          qe,
			PublicParamsFetcher: publicParamsFetcher,
			PPLabel:             fabtoken.PublicParameters,
		}),
		&fabtoken.VaultTokenLoader{TokenVault: qe},
		qe,
		identity.NewProvider(
			sp,
			map[driver.IdentityUsage]identity.Mapper{
				driver.IssuerRole:  fabric.NewMapper(networkID, fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, networkID).LocalMembership()),
				driver.AuditorRole: fabric.NewMapper(networkID, fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, networkID).LocalMembership()),
				driver.OwnerRole:   fabric.NewMapper(networkID, fabric.X509MSPIdentity, nodeIdentity, fabric2.GetFabricNetworkService(sp, networkID).LocalMembership()),
			},
		),
		fabtoken.NewDeserializer(),
	), nil
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	return fabtoken.NewValidator(params.(*fabtoken.PublicParams), fabtoken.NewDeserializer()), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	return fabtoken.NewPublicParamsManagerFromParams(params.(*fabtoken.PublicParams)), nil
}

func init() {
	core.Register(fabtoken.PublicParameters, &Driver{})
}
