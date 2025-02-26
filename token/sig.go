/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
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
	deserializer     driver.Deserializer
	identityProvider driver.IdentityProvider
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
	return s.identityProvider.GetSigner(id)
}

// RegisterSigner registers the pair (signer, verifier) bound to the given identity
func (s *SignatureService) RegisterSigner(identity Identity, signer Signer, verifier Verifier) error {
	return s.identityProvider.RegisterSigner(identity, signer, verifier, nil)
}

// AreMe returns the hashes of the passed identities that have a signer registered before
func (s *SignatureService) AreMe(identities ...Identity) []string {
	return s.identityProvider.AreMe(identities...)
}

// IsMe returns true if for the given identity there is a signer registered
func (s *SignatureService) IsMe(party Identity) bool {
	return s.identityProvider.IsMe(party)
}

// GetAuditInfo returns the audit infor
func (s *SignatureService) GetAuditInfo(ids ...Identity) ([][]byte, error) {
	result := make([][]byte, 0, len(ids))
	for _, id := range ids {
		auditInfo, err := s.identityProvider.GetAuditInfo(id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get audit info for identity [%s]", id)
		}
		result = append(result, auditInfo)
	}
	return result, nil
}
