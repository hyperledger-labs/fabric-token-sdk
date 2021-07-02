/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func (s *service) NewCertificationRequest(ids []*token2.Id) ([]byte, error) {
	return nil, nil
}

func (s *service) Certify(wallet driver.CertifierWallet, ids []*token2.Id, tokens [][]byte, request []byte) ([][]byte, error) {
	return nil, nil
}

func (s *service) VerifyCertifications(ids []*token2.Id, certifications [][]byte) error {
	return nil
}
