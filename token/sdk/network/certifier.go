/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier"
)

type CertificationClientProvider struct {
}

func NewCertificationClientProvider() *CertificationClientProvider {
	return &CertificationClientProvider{}
}

func (c *CertificationClientProvider) New(tms *token.ManagementService) (driver.CertificationClient, error) {
	return certifier.NewCertificationClient(tms)
}
