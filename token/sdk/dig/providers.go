/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"go.uber.org/dig"
)

func NewDriverHolder(in struct {
	dig.In
	Drivers        []dbdriver.NamedDriver `group:"token-db-drivers"`
	ConfigProvider driver2.ConfigService
}) *db.DriverHolder {
	return db.NewDriverHolder(in.ConfigProvider, in.Drivers...)
}

func newTokenDriverService(in struct {
	dig.In
	Drivers []core.NamedFactory[driver.Driver] `group:"token-drivers"`
}) *core.TokenDriverService {
	return core.NewTokenDriverService(in.Drivers)
}
