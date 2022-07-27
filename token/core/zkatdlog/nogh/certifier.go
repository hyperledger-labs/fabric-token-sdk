/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	api3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// NewCertificationRequest creates a request to certify the tokens identified by the passed identifiers
// zkatdlog does not make use of the certification service
func (s *Service) NewCertificationRequest(ids []*token3.ID) ([]byte, error) {
	return nil, nil
}

// Certify returns an array of serialized certifications, such that the i^th certification asserts that
// the i^th passed token corresponds to the token associated with the i^th passed identifier
// zkatdlog does not make use of the certification service
func (s *Service) Certify(wallet api3.CertifierWallet, ids []*token3.ID, tokens [][]byte, request []byte) ([][]byte, error) {
	return nil, nil
}

// VerifyCertifications checks if the passed certifications are valid with respect to the tokens associated
// with the passed identifiers
// If not, VerifyCertifications returns an error
// zkatdlog does not make use of the certification service
func (s *Service) VerifyCertifications(ids []*token3.ID, certifications [][]byte) error {
	return nil
}
