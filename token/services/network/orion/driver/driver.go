/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/pkg/errors"
)

type Driver struct {
}

func (d *Driver) New(sp view.ServiceProvider, network, channel string) (driver.Network, error) {
	n := orion.GetOrionNetworkService(sp, network)
	if n == nil {
		return nil, errors.Errorf("network %s not found", network)
	}

	return orion2.NewNetwork(
		sp,
		view.GetIdentityProvider(sp),
		n,
	), nil
}

func init() {
	network.Register("orion", &Driver{})
}
