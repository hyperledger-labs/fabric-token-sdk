/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"go.uber.org/dig"
)

func NewNetwork(in struct {
	dig.In
	SP      view2.ServiceProvider
	Drivers []driver.NamedDriver `group:"network-drivers"`
}) *network.Provider {
	return network.NewProvider(in.SP, in.Drivers)
}
