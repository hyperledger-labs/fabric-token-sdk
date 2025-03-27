/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
)

type (
	WalletID              = identity.WalletID
	IdentityConfiguration = driver.IdentityConfiguration
	WalletDB              = identity.WalletDB
	IdentityDB            = identity.IdentityDB
)

// IdentityDBDriver is the interface for an identity database driver
type IdentityDBDriver interface {
	// OpenWalletDB opens a connection to the wallet DB
	OpenWalletDB(cp ConfigProvider, tmsID token.TMSID) (WalletDB, error)
	// OpenIdentityDB opens a connection to the identity DB
	OpenIdentityDB(cp ConfigProvider, tmsID token.TMSID) (IdentityDB, error)
}
