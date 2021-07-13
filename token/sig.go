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
	s driver.SigService
}

func (s *SignatureService) GetVerifier(id view.Identity) (Verifier, error) {
	return s.s.GetVerifier(id)
}

func (s *SignatureService) GetSigner(id view.Identity) (Signer, error) {
	return s.s.GetSigner(id)
}

func (s *SignatureService) RegisterSigner(identity view.Identity, signer Signer, verifier Verifier) error {
	return s.s.RegisterSigner(identity, signer, verifier)
}
