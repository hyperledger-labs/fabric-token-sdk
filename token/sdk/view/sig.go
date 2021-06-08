/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package view

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
)

type SigService interface {
	GetVerifier(id view.Identity) (view2.Verifier, error)
	GetSigner(id view.Identity) (view2.Signer, error)
}

type SigServiceWrapper struct {
	s SigService
}

func NewSigServiceWrapper(s SigService) *SigServiceWrapper {
	return &SigServiceWrapper{s: s}
}

func (s *SigServiceWrapper) GetVerifier(id view.Identity) (api.Verifier, error) {
	return s.s.GetVerifier(id)
}

func (s *SigServiceWrapper) GetSigner(id view.Identity) (api.Signer, error) {
	return s.s.GetSigner(id)
}
