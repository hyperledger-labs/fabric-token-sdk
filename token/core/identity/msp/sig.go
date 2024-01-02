/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	api2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type SigService struct {
	*view.SigService
}

func NewSigService(sigService *view.SigService) *SigService {
	return &SigService{SigService: sigService}
}

func (s *SigService) RegisterSigner(identity view2.Identity, signer driver.Signer, verifier driver.Verifier) error {
	return s.SigService.RegisterSigner(identity, signer, verifier)
}

func (s *SigService) RegisterVerifier(identity view2.Identity, verifier api2.Verifier) error {
	return s.SigService.RegisterVerifier(identity, verifier)
}
