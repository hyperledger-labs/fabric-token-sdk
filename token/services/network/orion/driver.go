/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/pkg/errors"
)

type Driver struct{}

func (d *Driver) New(sp token.ServiceProvider, network, channel string) (driver.Network, error) {
	n, err := orion.GetOrionNetworkService(sp, network)
	if err != nil {
		return nil, errors.WithMessagef(err, "network [%s] not found", network)
	}
	m, err := vault.GetProvider(sp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault manager")
	}
	ttxdbProvider, err := ttxdb.GetProvider(sp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttxdb manager")
	}
	auditDBProvider, err := auditdb.GetProvider(sp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get audit db provider")
	}
	configProvider := view.GetConfigService(sp)
	enabled, err := IsCustodian(configProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed checking if custodian is enabled")
	}
	logger.Infof("Orion Custodian enabled: %t", enabled)
	if enabled {
		if err := InstallViews(view.GetRegistry(sp)); err != nil {
			return nil, errors.WithMessagef(err, "failed installing views")
		}
	}
	cs, err := config.GetService(sp)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get config service")
	}
	return NewNetwork(
		sp,
		view.GetIdentityProvider(sp),
		n,
		m.Vault,
		cs,
		common.NewAcceptTxInDBFilterProvider(ttxdbProvider, auditDBProvider),
	), nil
}

func init() {
	network.Register("orion", &Driver{})
}
