/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type CertificationClient interface {
	IsCertified(id *token2.ID) bool
	RequestCertification(ids ...*token2.ID) error
}

type CertificationService interface {
	Start() error
}

type Driver interface {
	NewCertificationClient(tms *token.ManagementService) (CertificationClient, error)
	NewCertificationService(tms *token.ManagementService, wallet string) (CertificationService, error)
}
