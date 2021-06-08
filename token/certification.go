/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type CertificationManager struct {
	c api.CertificationService
}

func (c *CertificationManager) NewCertificationRequest(ids []*token2.Id) ([]byte, error) {
	return c.c.NewCertificationRequest(ids)
}

func (c *CertificationManager) Certify(wallet *CertifierWallet, ids []*token2.Id, tokens [][]byte, request []byte) ([][]byte, error) {
	return c.c.Certify(wallet.w, ids, tokens, request)
}

func (c *CertificationManager) VerifyCertifications(ids []*token2.Id, certifications [][]byte) error {
	return c.c.VerifyCertifications(ids, certifications)
}

type CertificationClient struct {
	cc api.CertificationClient
}

func (c *CertificationClient) IsCertified(id *token2.Id) bool {
	return c.cc.IsCertified(id)
}

func (c *CertificationClient) RequestCertification(ids ...*token2.Id) error {
	return c.cc.RequestCertification(ids...)
}
