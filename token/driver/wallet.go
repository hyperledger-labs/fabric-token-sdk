/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// ListTokensOptions contains options that can be used to list tokens from a wallet
type ListTokensOptions struct {
	// TokenType is the type of token to list
	TokenType string
}

// Wallet models a generic walleet
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
	GetRecipientIdentity() (view.Identity, error)

	// GetAuditInfo returns auditing information for the passed identity
	GetAuditInfo(id view.Identity) ([]byte, error)

	// ListTokens returns the list of unspent tokens owned by this wallet filtered using the passed options.
	ListTokens(opts *ListTokensOptions) (*token.UnspentTokens, error)

	// ListTokensIterator returns an iterator of unspent tokens owned by this wallet filtered using the passed options.
	ListTokensIterator(opts *ListTokensOptions) (UnspentTokensIterator, error)

	// GetTokenMetadata returns any information needed to implement the transfer
	GetTokenMetadata(id view.Identity) ([]byte, error)

	// EnrollmentID returns the enrollment ID of the owner wallet
	EnrollmentID() string
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

// WalletService models the wallet service that handles issuer, recipient, auditor and certifier wallets
type WalletService interface {
	// RegisterRecipientIdentity registers the passed recipient identity together with the associated audit information
	RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error

	// GetAuditInfo retrieves the audit information for the passed identity
	GetAuditInfo(id view.Identity) ([]byte, error)

	// GetEnrollmentID extracts the enrollment id from the passed audit information
	GetEnrollmentID(auditInfo []byte) (string, error)

	// Wallet returns the wallet bound to the passed identity, if any is available
	Wallet(identity view.Identity) Wallet

	// RegisterOwnerWallet registers an owner wallet
	RegisterOwnerWallet(id string, path string) error

	// OwnerWallet returns an instance of the OwnerWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	OwnerWallet(id string) OwnerWallet

	// OwnerWalletByIdentity returns the OwnerWallet the passed identity belongs to.
	OwnerWalletByIdentity(identity view.Identity) OwnerWallet

	// IssuerWallet returns an instance of the IssuerWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	IssuerWallet(id string) IssuerWallet

	// IssuerWalletByIdentity returns an instance of the IssuerWallet interface that contains the passed identity.
	IssuerWalletByIdentity(identity view.Identity) IssuerWallet

	// AuditorWalletByIdentity returns an instance of the AuditorWallet interface that contains the passed identity.
	AuditorWalletByIdentity(identity view.Identity) AuditorWallet

	// AuditorWallet returns an instance of the AuditorWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	AuditorWallet(id string) AuditorWallet

	// CertifierWallet returns an instance of the CertifierWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	CertifierWallet(id string) CertifierWallet

	// CertifierWalletByIdentity returns an instance of the CertifierWallet interface that contains the passed identity.
	CertifierWalletByIdentity(identity view.Identity) CertifierWallet

	// SpentIDs returns the spend ids for the passed token ids
	SpentIDs(ids ...*token.ID) ([]string, error)
}

// Matcher models a matcher that can be used to match identities
type Matcher interface {
	// Match returns true if the passed identity matches this matcher
	Match([]byte) error
}

// Deserializer models the deserializer of owner, issuer, and auditor identities to
// get signature verifiers
type Deserializer interface {
	// GetOwnerVerifier returns the verifier associated to the passed owner identity
	GetOwnerVerifier(id view.Identity) (Verifier, error)
	// GetIssuerVerifier returns the verifier associated to the passed issuer identity
	GetIssuerVerifier(id view.Identity) (Verifier, error)
	// GetAuditorVerifier returns the verifier associated to the passed auditor identity
	GetAuditorVerifier(id view.Identity) (Verifier, error)
	// GetOwnerMatcher returns an identity matcher for the passed identity audit data.
	GetOwnerMatcher(auditData []byte) (Matcher, error)
}
