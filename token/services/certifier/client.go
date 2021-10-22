/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package certifier

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type CertificationClient struct {
	c driver.CertificationClient
}

func NewCertificationClient(sp view.ServiceProvider, network, channel, namespace, driver string) (*CertificationClient, error) {
	d, ok := drivers[driver]
	if !ok {
		return nil, errors.Errorf("certifier driver [%s] not found", driver)
	}
	c, err := d.NewCertificationClient(sp, network, channel, namespace)
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
