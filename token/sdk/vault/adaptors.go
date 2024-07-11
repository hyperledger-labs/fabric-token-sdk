/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

type Vault struct {
	*network.TokenVault
}

func (v *Vault) QueryEngine() driver.QueryEngine {
	return v.TokenVault.QueryEngine()
}

func (v *Vault) CertificationStorage() driver.CertificationStorage {
	return v.TokenVault.CertificationStorage()
}

type ProviderAdaptor struct {
	Provider *network.Provider
}

func (p *ProviderAdaptor) Vault(networkID string, channel string, namespace string) (driver.Vault, error) {
	net, err := p.Provider.GetNetwork(networkID, channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}
	v, err := net.TokenVault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault for [%s:%s:%s]", networkID, channel, namespace)
	}
	return &Vault{TokenVault: v}, nil
}

type PublicParamsProvider struct {
	Provider *network.Provider
}

func (p *PublicParamsProvider) PublicParams(networkID string, channel string, namespace string) ([]byte, error) {
	net, err := p.Provider.GetNetwork(networkID, channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}
	v, err := net.TokenVault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault for [%s:%s:%s]", networkID, channel, namespace)
	}
	return v.QueryEngine().PublicParams()
}
