/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package dummy

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	ZKATDLog = "zkatdlog"
	FabToken = "fabtoken"
)

type CertificationClient struct{}

func (c *CertificationClient) IsCertified(id *token2.ID) bool {
	return true
}

func (c *CertificationClient) RequestCertification(ids ...*token2.ID) error {
	return nil
}

func (c *CertificationClient) Start() error {
	return nil
}

type CertificationService struct{}

func (c *CertificationService) Start() error {
	return nil
}

type Driver struct{}

func NewDriver() *Driver {
	return &Driver{}
}

func (d *Driver) NewCertificationClient(sp view2.ServiceProvider, network, channel, namespace string) (driver.CertificationClient, error) {
	return &CertificationClient{}, nil
}

func (d *Driver) NewCertificationService(sp view2.ServiceProvider, network, channel, namespace, wallet string) (driver.CertificationService, error) {
	return &CertificationService{}, nil
}

func init() {
	certifier.Register(FabToken, &Driver{})
	certifier.Register(ZKATDLog, &Driver{})
}
