/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certifier

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type CertificationClient struct {
	c driver.CertificationClient
}

func NewCertificationClient(tms *token.ManagementService) (*CertificationClient, error) {
	driver := tms.PublicParametersManager().PublicParameters().CertificationDriver()
	d, ok := drivers[driver]
	if !ok {
		return nil, errors.Errorf("certifier driver [%s] not found", driver)
	}
	c, err := d.NewCertificationClient(tms)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating certification manager with driver [%s]", driver)
	}
	return &CertificationClient{c: c}, nil
}

func (c *CertificationClient) IsCertified(id *token2.ID) bool {
	return c.c.IsCertified(id)
}

func (c *CertificationClient) RequestCertification(ids ...*token2.ID) error {
	return c.c.RequestCertification(ids...)
}
