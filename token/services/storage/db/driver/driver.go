/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

type NamedDriver = driver2.NamedDriver[Driver]

type Config = driver.Config

type Driver interface {
	NewTokenLock(driver.PersistenceName, ...string) (TokenLockStore, error)

	NewWallet(driver.PersistenceName, ...string) (WalletStore, error)

	NewIdentity(driver.PersistenceName, ...string) (IdentityStore, error)

	// NewIdentityNotifier returns a new IdentityNotifier for the given persistence name and params.
	NewIdentityNotifier(driver.PersistenceName, ...string) (idriver.IdentityConfigurationNotifier, error)

	NewToken(driver.PersistenceName, ...string) (TokenStore, error)

	NewTokenNotifier(driver.PersistenceName, ...string) (TokenNotifier, error)

	NewAuditTransaction(driver.PersistenceName, ...string) (AuditTransactionStore, error)

	NewOwnerTransaction(driver.PersistenceName, ...string) (TokenTransactionStore, error)

	NewKeyStore(name driver.PersistenceName, params ...string) (KeyStore, error)
}
