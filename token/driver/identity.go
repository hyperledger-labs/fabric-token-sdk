/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

// Identity represents a generic identity
type Identity = view.Identity

//go:generate counterfeiter -o mock/ip.go -fake-name IdentityProvider . IdentityProvider

// IdentityProvider manages identity-related concepts like signature signers, verifiers, audit information, and so on.
type IdentityProvider interface {
	// RegisterRecipientData stores the passed recipient data
	RegisterRecipientData(ctx context.Context, data *RecipientData) error

	// GetAuditInfo returns the audit information associated to the passed identity, nil otherwise
	GetAuditInfo(ctx context.Context, identity Identity) ([]byte, error)

	// GetSigner returns a Signer for passed identity.
	GetSigner(ctx context.Context, identity Identity) (Signer, error)

	// RegisterVerifier registers a Verifier for passed identity.
	RegisterVerifier(ctx context.Context, identity Identity, v Verifier) error

	// RegisterSigner registers a Signer and a Verifier for passed identity.
	RegisterSigner(ctx context.Context, identity Identity, signer Signer, verifier Verifier, signerInfo []byte, ephemeral bool) error

	// AreMe returns the hashes of the passed identities that have a signer registered before
	AreMe(ctx context.Context, identities ...Identity) []string

	// IsMe returns true if a signer was ever registered for the passed identity
	IsMe(ctx context.Context, party Identity) bool

	// GetEnrollmentID extracts the enrollment ID from the passed audit info
	GetEnrollmentID(ctx context.Context, identity Identity, auditInfo []byte) (string, error)

	// GetRevocationHandler extracts the revocation handler from the passed audit info
	GetRevocationHandler(ctx context.Context, identity Identity, auditInfo []byte) (string, error)

	// GetEIDAndRH returns both enrollment ID and revocation handle
	GetEIDAndRH(ctx context.Context, identity Identity, auditInfo []byte) (string, string, error)

	// Bind binds longTerm to the passed ephemeral identity. The same signer, verifier, and audit of the long term
	// identity is associated to id, if copyAll is true.
	Bind(ctx context.Context, longTerm Identity, ephemeralIdentities ...Identity) error

	// RegisterRecipientIdentity register the passed identity as a third-party recipient identity.
	RegisterRecipientIdentity(ctx context.Context, id Identity) error
}
