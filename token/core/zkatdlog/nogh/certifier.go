/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	api3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *service) NewCertificationRequest(ids []*token3.Id) ([]byte, error) {
	return nil, nil
}

func (s *service) Certify(wallet api3.CertifierWallet, ids []*token3.Id, tokens [][]byte, request []byte) ([][]byte, error) {
	return nil, nil
}

func (s *service) VerifyCertifications(ids []*token3.Id, certifications [][]byte) error {
	return nil
}
