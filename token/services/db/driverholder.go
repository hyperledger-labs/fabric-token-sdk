/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/multiplexed"
)

func NewDriverHolder(cp driver2.ConfigService, drivers multiplexed.Driver) *DriverHolder {
	return &DriverHolder{drivers: drivers, config: config.NewService(cp)}
}

type DriverHolder struct {
	drivers multiplexed.Driver
	config  *config.Service
}

func (h *DriverHolder) NewTokenLockManager() *Manager[driver.TokenLockStore] {
	return newManager(h.config, "tokenlockdb.persistence", h.drivers.NewTokenLock)
}

func (h *DriverHolder) NewWalletManager() *Manager[driver.WalletStore] {
	return newManager(h.config, "identitydb.persistence", h.drivers.NewWallet)
}

func (h *DriverHolder) NewIdentityManager() *Manager[driver.IdentityStore] {
	return newManager(h.config, "identitydb.persistence", h.drivers.NewIdentity)
}

func (h *DriverHolder) NewTokenManager() *Manager[driver.TokenStore] {
	return newManager(h.config, "tokendb.persistence", h.drivers.NewToken)
}

func (h *DriverHolder) NewTokenNotifierManager() *Manager[driver.TokenNotifier] {
	return newManager(h.config, "tokendb.persistence", h.drivers.NewTokenNotifier)
}

func (h *DriverHolder) NewAuditTransactionManager() *Manager[driver.AuditTransactionStore] {
	return newManager(h.config, "auditdb.persistence", h.drivers.NewAuditTransaction)
}

func (h *DriverHolder) NewOwnerTransactionManager() *Manager[driver.TokenTransactionStore] {
	return newManager(h.config, "ttxdb.persistence", h.drivers.NewOwnerTransaction)
}
