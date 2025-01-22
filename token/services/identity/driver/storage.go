/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type Keystore interface {
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
}

type StorageProvider interface {
	OpenWalletDB(tmsID token.TMSID) (driver.WalletDB, error)
	OpenIdentityDB(tmsID token.TMSID) (driver.IdentityDB, error)
	NewKeystore() (Keystore, error)
}
