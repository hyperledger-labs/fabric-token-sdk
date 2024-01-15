/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certification

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type Storage = driver.CertificationStorage

type StorageProvider interface {
	NewStorage(tmsID token.TMSID) (Storage, error)
}

var (
	storageProviderType = reflect.TypeOf((*StorageProvider)(nil))
)

// GetStorageProvider returns the registered instance of StorageProvider from the passed service provider
func GetStorageProvider(sp view.ServiceProvider) (StorageProvider, error) {
	s, err := sp.GetService(storageProviderType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token vault provider")
	}
	return s.(StorageProvider), nil
}
