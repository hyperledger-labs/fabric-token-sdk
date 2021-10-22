/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *Service) NewCertificationRequest(ids []*token2.ID) ([]byte, error) {
	return nil, nil
}

func (s *Service) Certify(wallet driver.CertifierWallet, ids []*token2.ID, tokens [][]byte, request []byte) ([][]byte, error) {
	return nil, nil
}

func (s *Service) VerifyCertifications(ids []*token2.ID, certifications [][]byte) error {
	return nil
}
