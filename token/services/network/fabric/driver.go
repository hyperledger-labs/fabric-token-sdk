/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	vault2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/pkg/errors"
)

type Driver struct {
	fnsProvider      *fabric.NetworkServiceProvider
	vaultProvider    vault.Provider
	tokensManager    *tokens.Manager
	configService    *config.Service
	viewManager      *view.Manager
	viewRegistry     driver2.Registry
	filterProvider   *common.AcceptTxInDBFilterProvider
	tmsProvider      *token.ManagementServiceProvider
	identityProvider *view.IdentityProvider
}

func NewDriver(
	fnsProvider *fabric.NetworkServiceProvider,
	vaultProvider *vault2.Provider,
	tokensManager *tokens.Manager,
	configService *config.Service,
	viewManager *view.Manager,
	viewRegistry driver2.Registry,
	filterProvider *common.AcceptTxInDBFilterProvider,
	tmsProvider *token.ManagementServiceProvider,
) driver.NamedDriver {
	return driver.NamedDriver{
		Name: "fabric",
		Driver: &Driver{
			fnsProvider:    fnsProvider,
			vaultProvider:  vaultProvider,
			tokensManager:  tokensManager,
			configService:  configService,
			viewManager:    viewManager,
			viewRegistry:   viewRegistry,
			filterProvider: filterProvider,
			tmsProvider:    tmsProvider,
		},
	}
}

func (d *Driver) New(network, channel string) (driver.Network, error) {
	fns, err := d.fnsProvider.FabricNetworkService(network)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric network [%s] not found", network)
	}
	ch, err := fns.Channel(channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric channel [%s:%s] not found", network, channel)
	}

	return NewNetwork(
		fns,
		ch,
		d.vaultProvider.Vault,
		d.configService,
		d.filterProvider,
		d.tokensManager,
		d.viewManager,
		d.tmsProvider,
		endorsement.NewServiceProvider(
			fns,
			d.configService,
			d.viewManager,
			d.viewRegistry,
			d.identityProvider,
			d.tmsProvider,
		),
	), nil
}
