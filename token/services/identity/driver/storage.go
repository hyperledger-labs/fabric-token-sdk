/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type Keystore interface {
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
}

type StorageProvider interface {
	WalletStore(tmsID token.TMSID) (WalletStoreService, error)
	IdentityStore(tmsID token.TMSID) (IdentityStoreService, error)
	Keystore() (Keystore, error)
}
