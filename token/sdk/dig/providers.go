/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/multiplexed"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"go.uber.org/dig"
)

// newMultiplexedDriver creates a multiplexed database driver from registered drivers.
// It aggregates all token database drivers and provides a unified interface.
func newMultiplexedDriver(in struct {
	dig.In
	Drivers        []dbdriver.NamedDriver `group:"token-db-drivers"`
	ConfigProvider driver2.ConfigService
}) multiplexed.Driver {
	return multiplexed.NewDriver(in.ConfigProvider, in.Drivers...)
}

// newTokenDriverService creates a token driver service from registered token drivers.
// It manages different token driver implementations (e.g., zkat, fabtoken).
func newTokenDriverService(in struct {
	dig.In
	Drivers []core.NamedFactory[driver.Driver] `group:"token-drivers"`
}) *core.TokenDriverService {
	return core.NewTokenDriverService(in.Drivers)
}

type TokensServiceManager struct {
	sm *tokens.ServiceManager
}

func NewTokensServiceManager(sm *tokens.ServiceManager) *TokensServiceManager {
	return &TokensServiceManager{sm: sm}
}

func (t *TokensServiceManager) ServiceByTMSId(tmsid token.TMSID) (dep.TokensService, error) {
	return t.sm.ServiceByTMSId(tmsid)
}

type StoreServiceManager struct {
	sm ttx.StoreServiceManager
}

func NewStoreServiceManager(sm ttx.StoreServiceManager) *StoreServiceManager {
	return &StoreServiceManager{sm: sm}
}

func (t *StoreServiceManager) StoreServiceByTMSId(tmsid token.TMSID) (dep.StoreService, error) {
	return t.sm.StoreServiceByTMSId(tmsid)
}
