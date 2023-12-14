/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws"
	"github.com/pkg/errors"
)

type Provider struct {
	sp view.ServiceProvider

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]vault.TokenVault
}

func NewProvider(sp view.ServiceProvider) *Provider {
	return &Provider{sp: sp, vaultCache: make(map[string]vault.TokenVault)}
}

func (v *Provider) Vault(network string, channel string, namespace string) (vault.TokenVault, error) {
	k := network + channel + namespace
	// Check cache
	v.vaultCacheLock.RLock()
	res, ok := v.vaultCache[k]
	v.vaultCacheLock.RUnlock()
	if ok {
		return res, nil
	}

	// lock
	v.vaultCacheLock.Lock()
	defer v.vaultCacheLock.Unlock()

	// check cache again
	res, ok = v.vaultCache[k]
	if ok {
		return res, nil
	}

	tokenStore, err := processor.NewCommonTokenStore(v.sp, token3.TMSID{
		Network:   network,
		Channel:   channel,
		Namespace: namespace,
	})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token store")
	}

	// Create new vault
	fns := fabric.GetFabricNetworkService(v.sp, network)
	if fns != nil {
		ch := fabric.GetChannel(v.sp, network, channel)
		res = rws.NewVault(
			v.sp,
			ch.Name(),
			namespace,
			fabric2.NewVault(ch, tokenStore),
		)
	} else {
		ons := orion.GetOrionNetworkService(v.sp, network)
		if ons == nil {
			return nil, errors.Errorf("cannot find network [%s]", network)
		}
		res = rws.NewVault(
			v.sp,
			"",
			namespace,
			orion2.NewVault(ons, tokenStore),
		)
	}

	// update cache
	v.vaultCache[k] = res

	return res, nil
}

type ProviderAdaptor struct {
	*Provider
}

func (v *ProviderAdaptor) Vault(network string, channel string, namespace string) (driver.Vault, error) {
	return v.Provider.Vault(network, channel, namespace)
}

type PublicParamsProvider struct {
	Provider *Provider
}

func (p *PublicParamsProvider) PublicParams(networkID string, channel string, namespace string) ([]byte, error) {
	v, err := p.Provider.Vault(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault for [%s:%s:%s]", networkID, channel, namespace)
	}
	return v.QueryEngine().PublicParams()
}
