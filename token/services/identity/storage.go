/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
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

var (
	storageProviderType = reflect.TypeOf((*StorageProvider)(nil))
)

// GetStorageProvider returns the registered instance of StorageProvider from the passed service provider
func GetStorageProvider(sp token.ServiceProvider) (StorageProvider, error) {
	s, err := sp.GetService(storageProviderType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token vault provider")
	}
	return s.(StorageProvider), nil
}
