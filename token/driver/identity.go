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

// IdentityInfo models a long-term identity inside the Identity Provider.
// An identity has an identifier (ID) and an Enrollment ID, unique identifier.
// An identity can be remote, meaning that the corresponding secret key is remotely available.
type IdentityInfo interface {
	// ID returns the identifier of the Identity
	ID() string
	// EnrollmentID returns the enrollment ID of the Identity
	EnrollmentID() string
	// Remote is true if this identity info refers to an identify whose corresponding secret key is not known, it is external/remote
	Remote() bool
	// Get returns the identity and it is audit info.
	// Get might return a different identity at each call depending on the implementation.
	Get() (view.Identity, []byte, error)
}

//go:generate counterfeiter -o mock/ip.go -fake-name IdentityProvider . IdentityProvider

// IdentityProvider manages identity-related concepts like signature signers, verifiers, audit information, and so on.
type IdentityProvider interface {
	// RegisterRecipientData stores the passed recipient data
	RegisterRecipientData(data *RecipientData) error

	// GetAuditInfo returns the audit information associated to the passed identity, nil otherwise
	GetAuditInfo(identity view.Identity) ([]byte, error)

	// GetSigner returns a Signer for passed identity.
	GetSigner(identity view.Identity) (Signer, error)

	// RegisterVerifier registers a Verifier for passed identity.
	RegisterVerifier(identity view.Identity, v Verifier) error

	// RegisterSigner registers a Signer and a Verifier for passed identity.
	RegisterSigner(identity view.Identity, signer Signer, verifier Verifier, signerInfo []byte) error

	// IsMe returns true if a signer was ever registered for the passed identity
	IsMe(party view.Identity) bool

	// GetEnrollmentID extracts the enrollment ID from the passed audit info
	GetEnrollmentID(auditInfo []byte) (string, error)

	// GetRevocationHandler extracts the revocation handler from the passed audit info
	GetRevocationHandler(auditInfo []byte) (string, error)

	// Bind binds id to the passed identity long term identity. The same signer, verifier, and audit of the long term
	// identity is associated to id.
	Bind(id view.Identity, longTerm view.Identity) error

	// RegisterRecipientIdentity register the passed identity as a third-party recipient identity.
	RegisterRecipientIdentity(id view.Identity) error
}
