/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certifier

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
)

type CertificationService struct {
	c driver.CertificationService
}

func NewCertificationService(sp view.ServiceProvider, network, channel, namespace, wallet, driver string) (*CertificationService, error) {
	d, ok := drivers[driver]
	if !ok {
		return nil, errors.Errorf("certifier driver [%s] not found", driver)
	}

	if len(network) == 0 {
		return nil, errors.Errorf("no network specified")
	}
	if len(channel) == 0 {
		return nil, errors.Errorf("no channel specified")
	}
	if len(namespace) == 0 {
		return nil, errors.Errorf("no namespace specified")
	}

	c, err := d.NewCertificationService(sp, network, channel, namespace, wallet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating certifier with driver [%s]", driver)
	}
	return &CertificationService{c: c}, nil
}

func (c *CertificationService) Start() error {
	return c.c.Start()
}
