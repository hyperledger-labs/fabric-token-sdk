/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package stub

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

type Driver struct{}

func (d *Driver) New(sp view.ServiceProvider, network, channel string) (driver.Network, error) {
	// instantiate vault

	return NewNetwork(sp, n, ch), nil
}

func init() {
	network.Register("stub", &Driver{})
}
