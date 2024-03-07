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
	ID   string
	Type string
	URL  string
}

type WalletDB interface {
	// StoreWalletID stores an entry for the passed wallet id
	StoreWalletID(wID WalletID) error
	// GetWalletID fetches a walletID that is bound to the identity passed
	GetWalletID(identity view.Identity) (WalletID, error)
	// GetWalletIDs fetches all walletID's that have been stored so far without duplicates
	GetWalletIDs() ([]WalletID, error)
	// StoreIdentity binds an identity to a walletID and its metadata
	StoreIdentity(identity view.Identity, wID WalletID, meta any) error
	// IdentityExists checks whether an identity-wallet binding has already been stored
	IdentityExists(identity view.Identity, wID WalletID) bool
	// LoadMeta returns the metadata stored for a specific identity
	LoadMeta(identity view.Identity, meta any) error
}

type IdentityDB interface {
	// AddConfiguration stores an identity and the path to the credentials relevant to this identity
	AddConfiguration(wp IdentityConfiguration) error
	// IteratorConfigurations returns an iterator to all configurations stored
	IteratorConfigurations(configurationType string) (Iterator[IdentityConfiguration], error)
	// StoreAuditInfo stores the passed audit info and bound it to the given identity
	StoreAuditInfo(id, info []byte) error
	// GetAuditInfo retrieves the audit info bounded to the given identity
	GetAuditInfo(id []byte) ([]byte, error)
	// StoreSignerInfo stores the passed signer info and bound it to the given identity
	StoreSignerInfo(id, info []byte) error
	// SignerInfoExists returns true if StoreSignerInfo was called on input the given identity
	SignerInfoExists(id []byte) (bool, error)
}

// IdentityDBDriver is the interface for an identity database driver
type IdentityDBDriver interface {
	// OpenWalletDB opens a connection to the wallet DB
	OpenWalletDB(sp view2.ServiceProvider, tmsID token.TMSID) (WalletDB, error)
	// OpenIdentityDB opens a connection to the identity DB
	OpenIdentityDB(sp view2.ServiceProvider, tmsID token.TMSID) (IdentityDB, error)
}
