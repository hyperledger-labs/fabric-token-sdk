/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
)

type Vault struct {
	driver2.TokenVault
}

func (v *Vault) QueryEngine() driver.QueryEngine {
	return v.TokenVault.QueryEngine()
}

func (v *Vault) CertificationStorage() driver.CertificationStorage {
	return v.TokenVault.CertificationStorage()
}

type ProviderAdaptor struct {
	Provider *Provider
}

func (p *ProviderAdaptor) Vault(networkID string, channel string, namespace string) (driver.Vault, error) {
	vault, err := p.Provider.Vault(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}
	return &Vault{TokenVault: vault}, nil
}

type PublicParamsProvider struct {
	Provider *Provider
}

func (p *PublicParamsProvider) PublicParams(networkID string, channel string, namespace string) ([]byte, error) {
	vault, err := p.Provider.Vault(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}
	return vault.QueryEngine().PublicParams()
}
