/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier"
)

// CertificationClientProvider creates certification clients for token management services.
type CertificationClientProvider struct {
}

// NewCertificationClientProvider creates a new certification client provider.
func NewCertificationClientProvider() *CertificationClientProvider {
	return &CertificationClientProvider{}
}

// New creates a certification client for the given token management service.
func (c *CertificationClientProvider) New(ctx context.Context, tms *token.ManagementService) (driver.CertificationClient, error) {
	return certifier.NewCertificationClient(ctx, tms)
}
