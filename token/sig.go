/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type SignatureService struct {
	deserializer driver.Deserializer
	ip           driver.IdentityProvider
}

func (s *SignatureService) AuditorVerifier(id view.Identity) (Verifier, error) {
	return s.deserializer.GetAuditorVerifier(id)
}

func (s *SignatureService) OwnerVerifier(id view.Identity) (Verifier, error) {
	return s.deserializer.GetOwnerVerifier(id)
}

func (s *SignatureService) IssuerVerifier(id view.Identity) (Verifier, error) {
	return s.deserializer.GetIssuerVerifier(id)
}

func (s *SignatureService) GetSigner(id view.Identity) (Signer, error) {
	return s.ip.GetSigner(id)
}

func (s *SignatureService) RegisterSigner(identity view.Identity, signer Signer, verifier Verifier) error {
	return s.ip.RegisterSigner(identity, signer, verifier)
}

func (s *SignatureService) IsMe(party view.Identity) bool {
	return s.ip.IsMe(party)
}
