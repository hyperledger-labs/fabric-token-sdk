/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	api3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *Service) NewCertificationRequest(ids []*token3.ID) ([]byte, error) {
	return nil, nil
}

func (s *Service) Certify(wallet api3.CertifierWallet, ids []*token3.ID, tokens [][]byte, request []byte) ([][]byte, error) {
	return nil, nil
}

func (s *Service) VerifyCertifications(ids []*token3.ID, certifications [][]byte) error {
	return nil
}
