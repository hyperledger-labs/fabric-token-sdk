/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// CertificationManager manages token certifications as described by the paper
// [`Privacy-preserving auditable token payments in a permissioned blockchain system`]('https://eprint.iacr.org/2019/1058.pdf')
type CertificationManager struct {
	c driver.CertificationService
}

// NewCertificationRequest creates a new certification request for the passed token ids
func (c *CertificationManager) NewCertificationRequest(ids []*token2.ID) ([]byte, error) {
	return c.c.NewCertificationRequest(ids)
}

// Certify uses the passed wallet to certify the passed token ids.
// Certify takes in input the certification request and the token representations as available on the ledger.
func (c *CertificationManager) Certify(wallet *CertifierWallet, ids []*token2.ID, tokens [][]byte, request []byte) ([][]byte, error) {
	return c.c.Certify(wallet.w, ids, tokens, request)
}

// VerifyCertifications verfiies the certification of the passed token ids.
func (c *CertificationManager) VerifyCertifications(ids []*token2.ID, certifications [][]byte) error {
	return c.c.VerifyCertifications(ids, certifications)
}

// CertificationClient is the client side of the certification process
type CertificationClient struct {
	cc driver.CertificationClient
}

// IsCertified returns true if the passed token id has been already certified, otherwise false
func (c *CertificationClient) IsCertified(id *token2.ID) bool {
	return c.cc.IsCertified(id)
}

// RequestCertification requests the certification of the passed token ids
func (c *CertificationClient) RequestCertification(ids ...*token2.ID) error {
	return c.cc.RequestCertification(ids...)
}
