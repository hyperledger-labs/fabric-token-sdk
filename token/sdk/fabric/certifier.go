/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier"
)

type CertificationClientProvider struct {
	sp view.ServiceProvider
}

func NewCertificationClientProvider(sp view.ServiceProvider) *CertificationClientProvider {
	return &CertificationClientProvider{sp: sp}
}

func (c *CertificationClientProvider) New(network string, channel string, namespace string, driver string) (api.CertificationClient, error) {
	return certifier.NewCertificationClient(c.sp, network, channel, namespace, driver)
}
