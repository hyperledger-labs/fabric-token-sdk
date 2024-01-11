/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"reflect"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

type WalletID = string

type Storage interface {
	StoreWalletID(wID WalletID) error
	GetWalletID(identity view.Identity) (WalletID, error)
	GetWalletIDs() ([]WalletID, error)
	StoreIdentity(identity view.Identity, wID WalletID, meta any) error
	IdentityExists(identity view.Identity, wID WalletID) bool
	LoadMeta(identity view.Identity, meta any) error
}

type StorageProvider interface {
	New(tmsID token.TMSID) (Storage, error)
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
