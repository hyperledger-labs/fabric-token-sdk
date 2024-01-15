/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"reflect"

	"github.com/IBM/idemix/bccsp/keystore"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

type Iterator[T any] interface {
	HasNext() bool
	Close() error
	Next() (T, error)
}

type WalletID = string

type WalletPath struct {
	ID   string
	Path string
}

type Storage interface {
	StoreWalletID(wID WalletID) error
	GetWalletID(identity view.Identity) (WalletID, error)
	GetWalletIDs() ([]WalletID, error)
	StoreIdentity(identity view.Identity, wID WalletID, meta any) error
	IdentityExists(identity view.Identity, wID WalletID) bool
	LoadMeta(identity view.Identity, meta any) error
}

type WalletPathStorage interface {
	AddWallet(id string, path string) error
	WalletPaths() (Iterator[WalletPath], error)
}

type StorageProvider interface {
	NewStorage(tmsID token.TMSID) (Storage, error)
	GetWalletPathStorage(id string) (WalletPathStorage, error)
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
