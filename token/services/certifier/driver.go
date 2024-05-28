/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package certifier

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
)

var holder = drivers.NewHolder[driver.Driver]()

func Register(name string, driver driver.Driver) { holder.Register(name, driver) }

func Drivers() []string { return holder.DriverNames() }
