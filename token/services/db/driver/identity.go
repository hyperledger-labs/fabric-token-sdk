/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type Iterator[T any] interface {
	HasNext() bool
	Close() error
	Next() (T, error)
}

type WalletID = string

type IdentityConfiguration struct {
	ID  string
	URL string
}

type WalletDB interface {
	StoreWalletID(wID WalletID) error
	GetWalletID(identity view.Identity) (WalletID, error)
	GetWalletIDs() ([]WalletID, error)
	StoreIdentity(identity view.Identity, wID WalletID, meta any) error
	IdentityExists(identity view.Identity, wID WalletID) bool
	LoadMeta(identity view.Identity, meta any) error
}

type IdentityDB interface {
	AddConfiguration(wp IdentityConfiguration) error
	IteratorConfigurations() (Iterator[IdentityConfiguration], error)
}

// IdentityDBDriver is the interface for an identity database driver
type IdentityDBDriver interface {
	OpenWalletDB(sp view2.ServiceProvider, tmsID token.TMSID) (WalletDB, error)
	OpenIdentityDB(sp view2.ServiceProvider, tmsID token.TMSID, id string) (IdentityDB, error)
}
