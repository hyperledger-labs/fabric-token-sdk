/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certifier

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
)

type CertificationService struct {
	c driver.CertificationService
}

func NewCertificationService(tms *token.ManagementService, wallet string) (*CertificationService, error) {
	driver := tms.PublicParametersManager().PublicParameters().CertificationDriver()
	d, ok := holder.Drivers[driver]
	if !ok {
		return nil, errors.Errorf("certifier driver [%s] not found", driver)
	}

	c, err := d.NewCertificationService(tms, wallet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating certifier with driver [%s]", driver)
	}
	return &CertificationService{c: c}, nil
}

func (c *CertificationService) Start() error {
	return c.c.Start()
}
