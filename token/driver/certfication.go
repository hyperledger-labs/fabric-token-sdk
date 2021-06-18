/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

type CertificationClient interface {
	IsCertified(id *token2.Id) bool
	RequestCertification(ids ...*token2.Id) error
}

type CertificationService interface {
	NewCertificationRequest(ids []*token2.Id) ([]byte, error)
	Certify(wallet CertifierWallet, ids []*token2.Id, tokens [][]byte, request []byte) ([][]byte, error)
	VerifyCertifications(ids []*token2.Id, certifications [][]byte) error
}
