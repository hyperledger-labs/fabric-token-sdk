/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type Iterator[T any] interface {
	HasNext() bool
	Close() error
	Next() (T, error)
}

type WalletID = string

type IdentityConfiguration struct {
	ID     string
	Type   string
	URL    string
	Config []byte
	Raw    []byte
}

type WalletDB interface {
	// GetWalletID fetches a walletID that is bound to the identity passed
	GetWalletID(identity token.Identity, roleID int) (WalletID, error)
	// GetWalletIDs fetches all walletID's that have been stored so far without duplicates
	GetWalletIDs(roleID int) ([]WalletID, error)
	// StoreIdentity binds an identity to a walletID and its metadata
	StoreIdentity(identity token.Identity, eID string, wID WalletID, roleID int, meta []byte) error
	// IdentityExists checks whether an identity-wallet binding has already been stored
	IdentityExists(identity token.Identity, wID WalletID, roleID int) bool
	// LoadMeta returns the metadata stored for a specific identity
	LoadMeta(identity token.Identity, wID WalletID, roleID int) ([]byte, error)
}

type IdentityDB interface {
	// AddConfiguration stores an identity and the path to the credentials relevant to this identity
	AddConfiguration(wp IdentityConfiguration) error
	// ConfigurationExists returns true if a configuration with the given id and type exists.
	ConfigurationExists(id, typ string) (bool, error)
	// IteratorConfigurations returns an iterator to all configurations stored
	IteratorConfigurations(configurationType string) (Iterator[IdentityConfiguration], error)
	// StoreIdentityData stores the passed identity and token information
	StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error
	// GetAuditInfo retrieves the audit info bounded to the given identity
	GetAuditInfo(id []byte) ([]byte, error)
	// GetTokenInfo returns the token information related to the passed identity
	GetTokenInfo(id []byte) ([]byte, []byte, error)
	// StoreSignerInfo stores the passed signer info and bound it to the given identity
	StoreSignerInfo(id, info []byte) error
	// SignerInfoExists returns true if StoreSignerInfo was called on input the given identity
	SignerInfoExists(id []byte) (bool, error)
	// GetSignerInfo returns the signer info bound to the given identity
	GetSignerInfo(id []byte) ([]byte, error)
}

// IdentityDBDriver is the interface for an identity database driver
type IdentityDBDriver interface {
	// OpenWalletDB opens a connection to the wallet DB
	OpenWalletDB(cp ConfigProvider, tmsID token.TMSID) (WalletDB, error)
	// OpenIdentityDB opens a connection to the identity DB
	OpenIdentityDB(cp ConfigProvider, tmsID token.TMSID) (IdentityDB, error)
}
