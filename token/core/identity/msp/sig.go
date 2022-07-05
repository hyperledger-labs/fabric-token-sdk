/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	api2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type SigService struct {
	sigService *view.SigService
}

func NewSigService(sigService *view.SigService) *SigService {
	return &SigService{sigService: sigService}
}

func (s *SigService) IsMe(identity view2.Identity) bool {
	return s.sigService.IsMe(identity)
}

func (s *SigService) RegisterSigner(identity view2.Identity, signer api2.Signer, verifier api2.Verifier) error {
	return s.sigService.RegisterSigner(identity, signer, verifier)
}
