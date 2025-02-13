/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type Manager struct {
	identityManager *db.Manager[driver.IdentityDB]
	walletManager   *db.Manager[driver.WalletDB]
}

func NewManager(dh *db.DriverHolder, keys ...string) *Manager {
	return &Manager{
		identityManager: dh.NewIdentityManager(keys...),
		walletManager:   dh.NewWalletManager(keys...),
	}
}

func (m *Manager) IdentityDBByTMSId(tmsID token.TMSID) (driver.IdentityDB, error) {
	return m.identityManager.DBByTMSId(tmsID)
}

func (m *Manager) WalletDBByTMSId(tmsID token.TMSID) (driver.WalletDB, error) {
	return m.walletManager.DBByTMSId(tmsID)
}
