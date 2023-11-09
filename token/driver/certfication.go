/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

type CertificationClient interface {
	IsCertified(id *token2.ID) bool
	RequestCertification(ids ...*token2.ID) error
}

type CertificationService interface {
	NewCertificationRequest(ids []*token2.ID) ([]byte, error)
	Certify(wallet CertifierWallet, ids []*token2.ID, tokens [][]byte, request []byte) ([][]byte, error)
	// VerifyCertifications verifies the validity of the certifications of each token indexed by its token-id.
	// The function returns the result of any processing of these certifications.
	// In the simplest case, VerifyCertifications returns the certifications got in input
	VerifyCertifications(ids []*token2.ID, certifications [][]byte) ([][]byte, error)
}
