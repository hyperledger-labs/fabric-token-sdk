/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

var holder = drivers.NewHolder[driver.Driver]()

func Register(name string, driver driver.Driver) { holder.Register(name, driver) }

func Drivers() []string { return holder.DriverNames() }
