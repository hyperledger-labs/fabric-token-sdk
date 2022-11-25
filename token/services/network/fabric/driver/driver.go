/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

type Driver struct {
}

func (d *Driver) New(sp view.ServiceProvider, network, channel string) (driver.Network, error) {
	n := fabric.GetFabricNetworkService(sp, network)
	if n == nil {
		return nil, errors.Errorf("fabric network %s not found", network)
	}
	ch, err := n.Channel(channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "fabric channel [%s:%s] not found", network, channel)
	}

	return fabric2.NewNetwork(sp, n, ch), nil
}

func init() {
	network.Register("fabric", &Driver{})
}
