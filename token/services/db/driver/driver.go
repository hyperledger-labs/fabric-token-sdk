/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
)

type NamedDriver = driver2.NamedDriver[Driver]

type Config = driver.Config

type Driver interface {
	NewTokenLock(context.Context, driver.PersistenceName, ...string) (TokenLockStore, error)

	NewWallet(context.Context, driver.PersistenceName, ...string) (WalletStore, error)

	NewIdentity(context.Context, driver.PersistenceName, ...string) (IdentityStore, error)

	NewToken(context.Context, driver.PersistenceName, ...string) (TokenStore, error)

	NewTokenNotifier(context.Context, driver.PersistenceName, ...string) (TokenNotifier, error)

	NewAuditTransaction(context.Context, driver.PersistenceName, ...string) (AuditTransactionStore, error)

	NewOwnerTransaction(context.Context, driver.PersistenceName, ...string) (TokenTransactionStore, error)
}
