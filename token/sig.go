/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
)

type SignatureService struct {
	s api.SigService
}

func (s *SignatureService) GetVerifier(id view.Identity) (Verifier, error) {
	return s.s.GetVerifier(id)
}

func (s *SignatureService) GetSigner(id view.Identity) (Signer, error) {
	return s.s.GetSigner(id)
}
