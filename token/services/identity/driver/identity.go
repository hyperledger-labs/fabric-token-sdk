/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type IdentityDescriptor struct {
	Identity  driver.Identity
	AuditInfo []byte

	Signer     driver.Signer
	SignerInfo []byte
	Verifier   driver.Verifier
}

type IdentityConfigurationIterator = iterators.Iterator[*IdentityConfiguration]

type WalletID = string

type WalletStoreService interface {
	// GetWalletID fetches a walletID that is bound to the identity passed
	GetWalletID(ctx context.Context, identity token.Identity, roleID int) (WalletID, error)
	// GetWalletIDs fetches all walletID's that have been stored so far without duplicates
	GetWalletIDs(ctx context.Context, roleID int) ([]WalletID, error)
	// StoreIdentity binds an identity to a walletID and its metadata
	StoreIdentity(ctx context.Context, identity token.Identity, eID string, wID WalletID, roleID int, meta []byte) error
	// IdentityExists checks whether an identity-wallet binding has already been stored
	IdentityExists(ctx context.Context, identity token.Identity, wID WalletID, roleID int) bool
	// LoadMeta returns the metadata stored for a specific identity
	LoadMeta(ctx context.Context, identity token.Identity, wID WalletID, roleID int) ([]byte, error)
	// Close closes the store
	Close() error
}

type IdentityStoreService interface {
	// AddConfiguration stores an identity and the path to the credentials relevant to this identity
	AddConfiguration(ctx context.Context, wp IdentityConfiguration) error
	// ConfigurationExists returns true if a configuration with the given id and type exists.
	ConfigurationExists(ctx context.Context, id, typ, url string) (bool, error)
	// IteratorConfigurations returns an iterator to all configurations stored
	IteratorConfigurations(ctx context.Context, configurationType string) (IdentityConfigurationIterator, error)
	// StoreIdentityData stores the passed identity and token information
	StoreIdentityData(ctx context.Context, id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error
	// GetAuditInfo retrieves the audit info bounded to the given identity
	GetAuditInfo(ctx context.Context, id []byte) ([]byte, error)
	// GetTokenInfo returns the token information related to the passed identity
	GetTokenInfo(ctx context.Context, id []byte) ([]byte, []byte, error)
	// StoreSignerInfo stores the passed signer info and bound it to the given identity
	StoreSignerInfo(ctx context.Context, id, info []byte) error
	// GetExistingSignerInfo returns the hashes of the identities for which StoreSignerInfo was called
	GetExistingSignerInfo(ctx context.Context, ids ...driver.Identity) ([]string, error)
	// SignerInfoExists returns true if StoreSignerInfo was called on input the given identity
	SignerInfoExists(ctx context.Context, id []byte) (bool, error)
	// GetSignerInfo returns the signer info bound to the given identity
	GetSignerInfo(ctx context.Context, id []byte) ([]byte, error)
	RegisterIdentityDescriptor(ctx context.Context, descriptor *IdentityDescriptor, alias driver.Identity) error
	// Close closes the store
	Close() error
}
