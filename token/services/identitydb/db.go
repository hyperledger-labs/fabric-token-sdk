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
	identityManager *db.Manager[driver.IdentityStore]
	walletManager   *db.Manager[driver.WalletStore]
}

func NewManager(dh *db.DriverHolder) *Manager {
	return &Manager{
		identityManager: dh.NewIdentityManager(),
		walletManager:   dh.NewWalletManager(),
	}
}

func (m *Manager) IdentityDBByTMSId(tmsID token.TMSID) (driver.IdentityStore, error) {
	return m.identityManager.DBByTMSId(tmsID)
}

func (m *Manager) WalletDBByTMSId(tmsID token.TMSID) (driver.WalletStore, error) {
	return m.walletManager.DBByTMSId(tmsID)
}
