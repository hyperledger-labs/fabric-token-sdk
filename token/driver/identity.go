/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

// IdentityRole is the role of an identity
type IdentityRole int

const (
	// IssuerRole is the role of an issuer
	IssuerRole = iota
	// AuditorRole is the role of an auditor
	AuditorRole
	// OwnerRole is the role of an owner
	OwnerRole
	// CertifierRole is the role of a certifier
	CertifierRole
)

// IdentityInfo models an Identity inside the Identity Provider
type IdentityInfo interface {
	// ID returns the ID of the Identity
	ID() string
	// EnrollmentID returns the enrollment ID of the Identity
	EnrollmentID() string
	// Get returns the identity and it is audit info.
	// Get might return a different identity at each call depending on the implementation.
	Get() (view.Identity, []byte, error)
}

// IdentityProvider handles the long-term identities on top of which wallets are defined.
type IdentityProvider interface {
	LookupIdentifier(role IdentityRole, v interface{}) (view.Identity, string, error)

	// GetIdentityInfo returns the long-term identity info associated to the passed id, nil if not found.
	GetIdentityInfo(role IdentityRole, id string) (IdentityInfo, error)

	// GetAuditInfo returns the audit information associated to the passed identity, nil otherwise
	GetAuditInfo(identity view.Identity) ([]byte, error)

	// GetSigner returns a Signer for passed identity.
	GetSigner(identity view.Identity) (Signer, error)

	// RegisterSigner registers a Signer and a Verifier for passed identity.
	RegisterSigner(identity view.Identity, signer Signer, verifier Verifier) error

	// IsMe returns true if a signer was ever registered for the passed identity
	IsMe(party view.Identity) bool

	// GetEnrollmentID extracts the enrollment ID from the passed audit info
	GetEnrollmentID(auditInfo []byte) (string, error)

	// Bind binds id to the passed identity long term identity. The same signer, verifier, and audit of the long term
	// identity is associated to id.
	Bind(id view.Identity, longTerm view.Identity) error

	// RegisterRecipientIdentity register the passed identity as a third-pary recipient identity.
	RegisterRecipientIdentity(id view.Identity) error

	// RegisterOwnerWallet registers the passed wallet as the owner wallet of the passed identity.
	RegisterOwnerWallet(id string, path string) error

	// RegisterIssuerWallet registers the passed wallet ad the issuer wallet of the passed identity.
	RegisterIssuerWallet(id string, path string) error
}
