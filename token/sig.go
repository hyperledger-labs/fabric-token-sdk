/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Verifier models a signature verifier
type Verifier interface {
	// Verify verifies the signature of a given message
	Verify(message, sigma []byte) error
}

// Signer models a signature signer
type Signer interface {
	// Sign signs message bytes and returns the signature or an error on failure.
	Sign(message []byte) ([]byte, error)
}

// SignatureService gives access to signature verifiers and signers bound to identities known by
// this service
type SignatureService struct {
	deserializer driver.Deserializer
	ip           driver.IdentityProvider
}

// AuditorVerifier returns a signature verifier for the given auditor identity
func (s *SignatureService) AuditorVerifier(id view.Identity) (Verifier, error) {
	return s.deserializer.GetAuditorVerifier(id)
}

// OwnerVerifier returns a signature verifier for the given owner identity
func (s *SignatureService) OwnerVerifier(id view.Identity) (Verifier, error) {
	return s.deserializer.GetOwnerVerifier(id)
}

// IssuerVerifier returns a signature verifier for the given issuer identity
func (s *SignatureService) IssuerVerifier(id view.Identity) (Verifier, error) {
	return s.deserializer.GetIssuerVerifier(id)
}

// GetSigner returns a signer bound to the given identity
func (s *SignatureService) GetSigner(id view.Identity) (Signer, error) {
	return s.ip.GetSigner(id)
}

// RegisterSigner registers the pair (signer, verifier) bound to the given identity
func (s *SignatureService) RegisterSigner(identity view.Identity, signer Signer, verifier Verifier) error {
	return s.ip.RegisterSigner(identity, signer, verifier)
}

// IsMe returns true if for the given identity there is a signer registered
func (s *SignatureService) IsMe(party view.Identity) bool {
	return s.ip.IsMe(party)
}
