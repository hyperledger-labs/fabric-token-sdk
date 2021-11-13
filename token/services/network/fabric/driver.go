/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

type Driver struct {
}

func (d *Driver) New(sp view.ServiceProvider, network, channel string) (driver.Network, error) {
	n := fabric.GetFabricNetworkService(sp, network)
	if n == nil {
		return nil, errors.Errorf("network %s not found", network)
	}
	ch, err := n.Channel(channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "channel [%s:%s] not found", network, channel)
	}

	return NewNetwork(sp, n, ch), nil
}

func init() {
	network.Register("fabric", &Driver{})
}
