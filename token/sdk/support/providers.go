/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/multiplexed"
	"go.uber.org/dig"
)

func NewDBMultiplexedDriver(in struct {
	dig.In
	Drivers        []dbdriver.NamedDriver `group:"token-db-drivers"`
	ConfigProvider driver2.ConfigService
}) multiplexed.Driver {
	return multiplexed.NewDriver(in.ConfigProvider, in.Drivers...)
}

func NewTokenDriverService(in struct {
	dig.In
	Drivers []core.NamedFactory[driver.Driver] `group:"token-drivers"`
}) *core.TokenDriverService {
	return core.NewTokenDriverService(in.Drivers)
}
