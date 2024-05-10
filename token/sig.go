/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Identity represents a generic identity
type Identity = driver.Identity

// Verifier models a signature verifier
type Verifier = driver.Verifier

// Signer models a signature signer
type Signer = driver.Signer

// SignatureService gives access to signature verifiers and signers bound to identities known by
// this service
type SignatureService struct {
	deserializer driver.Deserializer
	ip           driver.IdentityProvider
}

// AuditorVerifier returns a signature verifier for the given auditor identity
func (s *SignatureService) AuditorVerifier(id Identity) (Verifier, error) {
	return s.deserializer.GetAuditorVerifier(id)
}

// OwnerVerifier returns a signature verifier for the given owner identity
func (s *SignatureService) OwnerVerifier(id Identity) (Verifier, error) {
	return s.deserializer.GetOwnerVerifier(id)
}

// IssuerVerifier returns a signature verifier for the given issuer identity
func (s *SignatureService) IssuerVerifier(id Identity) (Verifier, error) {
	return s.deserializer.GetIssuerVerifier(id)
}

// GetSigner returns a signer bound to the given identity
func (s *SignatureService) GetSigner(id Identity) (Signer, error) {
	return s.ip.GetSigner(id)
}

// RegisterSigner registers the pair (signer, verifier) bound to the given identity
func (s *SignatureService) RegisterSigner(identity Identity, signer Signer, verifier Verifier) error {
	return s.ip.RegisterSigner(identity, signer, verifier, nil)
}

// IsMe returns true if for the given identity there is a signer registered
func (s *SignatureService) IsMe(party Identity) bool {
	return s.ip.IsMe(party)
}
