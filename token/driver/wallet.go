/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// RecipientData contains information about the identity of a token owner
type RecipientData struct {
	// Identity is the identity of the token owner
	Identity view.Identity
	// AuditInfo contains private information Identity
	AuditInfo []byte
	// TokenMetadata contains public information related to the token to be assigned to this Recipient.
	TokenMetadata []byte
	// TokenMetadataAuditInfo contains private information TokenMetadata
	TokenMetadataAuditInfo []byte
}

// ListTokensOptions contains options that can be used to list tokens from a wallet
type ListTokensOptions struct {
	// TokenType is the type of token to list
	TokenType string
}

// Wallet models a generic wallet
type Wallet interface {
	// ID returns the ID of this wallet
	ID() string

	// Contains returns true if the passed identity belongs to this wallet
	Contains(identity view.Identity) bool

	// ContainsToken returns true if the passed token is owned by this wallet
	ContainsToken(token *token.UnspentToken) bool

	// GetSigner returns the Signer bound to the passed identity
	GetSigner(identity view.Identity) (Signer, error)
}

// OwnerWallet models the wallet of a token recipient.
type OwnerWallet interface {
	Wallet

	// GetRecipientIdentity returns a recipient identity.
	// Depending on the underlying wallet implementation, this can be a long-term or ephemeral identity.
	// Using the returned identity as an index, one can retrieve the following information:
	// - Identity audit info via GetAuditInfo;
	// - TokenMetadata via GetTokenMetadata;
	// - TokenIdentityMetadata via GetTokenMetadataAuditInfo.
	GetRecipientIdentity() (view.Identity, error)

	// GetAuditInfo returns auditing information for the passed identity
	GetAuditInfo(id view.Identity) ([]byte, error)

	// GetTokenMetadata returns the public information related to the token to be assigned to passed recipient identity.
	GetTokenMetadata(id view.Identity) ([]byte, error)

	// GetTokenMetadataAuditInfo returns private information about the token metadata assigned to the passed recipient identity.
	GetTokenMetadataAuditInfo(id view.Identity) ([]byte, error)

	// ListTokens returns the list of unspent tokens owned by this wallet filtered using the passed options.
	ListTokens(opts *ListTokensOptions) (*token.UnspentTokens, error)

	// ListTokensIterator returns an iterator of unspent tokens owned by this wallet filtered using the passed options.
	ListTokensIterator(opts *ListTokensOptions) (UnspentTokensIterator, error)

	// EnrollmentID returns the enrollment ID of the owner wallet
	EnrollmentID() string

	// RegisterRecipient register the given recipient data
	RegisterRecipient(data *RecipientData) error

	// Remote returns true if this wallet is verify only, meaning that the corresponding secret key is external to this wallet
	Remote() bool
}

// IssuerWallet models the wallet of an issuer
type IssuerWallet interface {
	Wallet

	// GetIssuerIdentity returns an issuer identity for the passed token type.
	// Depending on the underlying wallet implementation, this can be a long-term or ephemeral identity.
	GetIssuerIdentity(tokenType string) (view.Identity, error)

	// HistoryTokens returns the list of tokens issued by this wallet filtered using the passed options.
	HistoryTokens(opts *ListTokensOptions) (*token.IssuedTokens, error)
}

// AuditorWallet models the wallet of an auditor
type AuditorWallet interface {
	Wallet

	// GetAuditorIdentity returns an auditor identity.
	// Depending on the underlying wallet implementation, this can be a long-term or ephemeral identity.
	GetAuditorIdentity() (view.Identity, error)
}

// CertifierWallet models the wallet of a certifier
type CertifierWallet interface {
	Wallet

	// GetCertifierIdentity returns a certifier identity.
	// Depending on the underlying wallet implementation, this can be a long-term or ephemeral identity.
	GetCertifierIdentity() (view.Identity, error)
}

type IdentityConfiguration struct {
	ID     string
	URL    string
	Config []byte
	Raw    []byte
}

// WalletLookupID defines the type of identifiers that can be used to retrieve a given wallet.
// It can be a string, as the name of the wallet, or an identity contained in that wallet.
// Ultimately, it is the token driver to decide which types are allowed.
type WalletLookupID = any

// WalletService models the wallet service that handles issuer, recipient, auditor and certifier wallets
type WalletService interface {
	// RegisterRecipientIdentity registers the passed recipient identity together with the associated audit information
	RegisterRecipientIdentity(data *RecipientData) error

	// GetAuditInfo retrieves the audit information for the passed identity
	GetAuditInfo(id view.Identity) ([]byte, error)

	// GetEnrollmentID extracts the enrollment id from the passed audit information
	GetEnrollmentID(auditInfo []byte) (string, error)

	// GetRevocationHandler extracts the revocation handler from the passed audit information
	GetRevocationHandler(auditInfo []byte) (string, error)

	// Wallet returns the wallet bound to the passed identity, if any is available
	Wallet(identity view.Identity) Wallet

	// RegisterOwnerIdentity registers an owner long-term identity
	RegisterOwnerIdentity(config IdentityConfiguration) error

	// RegisterIssuerIdentity registers an issuer long-term wallet
	RegisterIssuerIdentity(config IdentityConfiguration) error

	// OwnerWalletIDs returns the list of owner wallet identifiers
	OwnerWalletIDs() ([]string, error)

	// OwnerWallet returns an instance of the OwnerWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	OwnerWallet(id WalletLookupID) (OwnerWallet, error)

	// IssuerWallet returns an instance of the IssuerWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	IssuerWallet(id WalletLookupID) (IssuerWallet, error)

	// AuditorWallet returns an instance of the AuditorWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	AuditorWallet(id WalletLookupID) (AuditorWallet, error)

	// CertifierWallet returns an instance of the CertifierWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	CertifierWallet(id WalletLookupID) (CertifierWallet, error)

	// SpentIDs returns the spend ids for the passed token ids
	SpentIDs(ids ...*token.ID) ([]string, error)
}

// Matcher models a matcher that can be used to match identities
type Matcher interface {
	// Match returns true if the passed identity matches this matcher
	Match([]byte) error
}

// AuditInfoProvider models a provider of audit information
type AuditInfoProvider interface {
	// GetAuditInfo returns the audit information for the given identity, if available.
	GetAuditInfo(identity view.Identity) ([]byte, error)
}

//go:generate counterfeiter -o mock/deserializer.go -fake-name Deserializer . Deserializer

// Deserializer models the deserializer of owner, issuer, and auditor identities to
// get signature verifiers
type Deserializer interface {
	// GetOwnerVerifier returns the verifier associated to the passed owner identity
	GetOwnerVerifier(id view.Identity) (Verifier, error)
	// GetIssuerVerifier returns the verifier associated to the passed issuer identity
	GetIssuerVerifier(id view.Identity) (Verifier, error)
	// GetAuditorVerifier returns the verifier associated to the passed auditor identity
	GetAuditorVerifier(id view.Identity) (Verifier, error)
	// GetOwnerMatcher returns an identity matcher for the passed identity audit data
	GetOwnerMatcher(auditData []byte) (Matcher, error)
	// Recipients returns the recipient identities from the given serialized representation
	Recipients(raw []byte) ([]view.Identity, error)
	// Match returns nil if the given identity matches the given audit information.
	// An error otherwise
	Match(identity view.Identity, info []byte) error
	// GetOwnerAuditInfo returns the audit information for each identity contained in the given serialized representation
	GetOwnerAuditInfo(raw []byte, p AuditInfoProvider) ([][]byte, error)
}

// Serializer models the serialization needs of the Token Service
type Serializer interface {
	// MarshalTokenRequestToSign marshals the to token request to a byte array representation on which a signature must be produced
	MarshalTokenRequestToSign(request *TokenRequest, meta *TokenRequestMetadata) ([]byte, error)
}
