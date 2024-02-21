/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package stub

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

var logger = flogging.MustGetLogger("token-sdk.network.stub")

type Driver struct{}

func (d *Driver) New(sp view.ServiceProvider, network, channel string) (driver.Network, error) {
	// TODO instantiate or inject vault

	return NewNetwork(network, channel, nil), nil
}

func init() {
	network.Register("stub", &Driver{})
}
