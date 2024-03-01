/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"reflect"

	"github.com/IBM/idemix/bccsp/keystore"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type StorageProvider interface {
	OpenWalletDB(tmsID token.TMSID) (driver.WalletDB, error)
	OpenIdentityDB(tmsID token.TMSID, id string) (driver.IdentityDB, error)
	NewKeystore() (keystore.KVS, error)
}

var (
	storageProviderType = reflect.TypeOf((*StorageProvider)(nil))
)

// GetStorageProvider returns the registered instance of StorageProvider from the passed service provider
func GetStorageProvider(sp view2.ServiceProvider) (StorageProvider, error) {
	s, err := sp.GetService(storageProviderType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token vault provider")
	}
	return s.(StorageProvider), nil
}
