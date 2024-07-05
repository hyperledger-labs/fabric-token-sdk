/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	vault2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/pkg/errors"
)

func NewDriver(
	fnsProvider *fabric.NetworkServiceProvider,
	vaultProvider *vault2.Provider,
	tokensManager *tokens.Manager,
	configService *config.Service,
	viewManager *view.Manager,
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
			filterProvider: filterProvider,
			tmsProvider:    tmsProvider,
		},
	}
}

type Driver struct {
	fnsProvider    *fabric.NetworkServiceProvider
	vaultProvider  vault.Provider
	tokensManager  *tokens.Manager
	configService  *config.Service
	viewManager    *view.Manager
	filterProvider *common.AcceptTxInDBFilterProvider
	tmsProvider    *token.ManagementServiceProvider
}

func (d *Driver) New(network, channel string) (driver.Network, error) {
	n, err := d.fnsProvider.FabricNetworkService(network)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric network [%s] not found", network)
	}
	ch, err := n.Channel(channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric channel [%s:%s] not found", network, channel)
	}

	return NewNetwork(
		d.viewManager,
		d.tmsProvider,
		n,
		ch,
		d.vaultProvider.Vault,
		d.configService,
		d.filterProvider,
		d.tokensManager,
	), nil
}
