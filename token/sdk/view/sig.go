/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package view

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type SignerService interface {
	GetSigner(id view.Identity) (view2.Signer, error)
	RegisterSigner(identity view.Identity, signer view2.Signer, verifier view2.Verifier) error
}

type SignerServiceWrapper struct {
	s SignerService
}

func NewSignerServiceWrapper(s SignerService) *SignerServiceWrapper {
	return &SignerServiceWrapper{s: s}
}

func (s *SignerServiceWrapper) RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error {
	return s.s.RegisterSigner(identity, signer, verifier)
}

func (s *SignerServiceWrapper) GetSigner(id view.Identity) (driver.Signer, error) {
	return s.s.GetSigner(id)
}
