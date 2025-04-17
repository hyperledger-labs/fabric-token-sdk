/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
)

type NamedDriver = driver2.NamedDriver[Driver]

type Config = driver.Config

type Driver interface {
	NewTokenLock(Config, ...string) (TokenLockDB, error)

	NewWallet(Config, ...string) (WalletDB, error)

	NewIdentity(Config, ...string) (IdentityDB, error)

	NewToken(Config, ...string) (TokenDB, error)

	NewTokenNotifier(Config, ...string) (TokenNotifier, error)

	NewAuditTransaction(Config, ...string) (AuditTransactionDB, error)

	NewOwnerTransaction(Config, ...string) (TokenTransactionDB, error)
}
