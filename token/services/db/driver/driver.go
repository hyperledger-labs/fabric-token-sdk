/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type NamedDriver = driver.NamedDriver[Driver]

type Config interface {
	// ID identities the TMS this configuration refers to.
	ID() driver2.TMSID
	// IsSet checks to see if the key has been set in any of the data locations
	IsSet(key string) bool
	// UnmarshalKey takes a single key and unmarshals it into a struct.
	// The key must be relative to the TMS this configuration refers to.
	UnmarshalKey(key string, rawVal interface{}) error
	// GetString returns the value associated with the key as a string
	GetString(key string) string
	// GetBool returns the value associated with the key as a bool
	GetBool(key string) bool
	// TranslatePath translates the passed path relative to the config path
	TranslatePath(path string) string
}

type Driver interface {
	NewTokenLock(opts common.Opts) (TokenLockDB, error)

	NewWallet(opts common.Opts) (WalletDB, error)

	NewIdentity(opts common.Opts) (IdentityDB, error)

	NewToken(opts common.Opts) (TokenDB, error)

	NewTokenNotifier(opts common.Opts) (TokenNotifier, error)

	NewAuditTransaction(opts common.Opts) (AuditTransactionDB, error)

	NewOwnerTransaction(opts common.Opts) (TokenTransactionDB, error)
}
