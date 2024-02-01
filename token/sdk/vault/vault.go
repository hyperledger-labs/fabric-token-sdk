/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
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

	storageProvider, err := certification.GetStorageProvider(v.sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get storage provider")
	}

	// Create new vault
	if fns := fabric.GetFabricNetworkService(v.sp, network); fns != nil {
		ch := fabric.GetChannel(v.sp, network, channel)
		tmsID := token3.TMSID{
			Network:   network,
			Channel:   ch.Name(),
			Namespace: namespace,
		}
		storage, err := storageProvider.NewStorage(tmsID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new storage")
		}
		res, err = rws.NewVault(storage, tmsID, fabric2.NewVault(ch, tokenStore))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new vault")
		}
	} else {
		ons := orion.GetOrionNetworkService(v.sp, network)
		if ons == nil {
			return nil, errors.Errorf("cannot find network [%s]", network)
		}
		tmsID := token3.TMSID{
			Network:   network,
			Channel:   "",
			Namespace: namespace,
		}
		storage, err := storageProvider.NewStorage(tmsID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new storage")
		}
		res, err = rws.NewVault(storage, tmsID, orion2.NewVault(ons, tokenStore))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new vault")
		}
	}

	// update cache
	v.vaultCache[k] = res

	return res, nil
}

type ProviderAdaptor struct {
	Provider *network.Provider
}

func (p *ProviderAdaptor) Vault(networkID string, channel string, namespace string) (driver.Vault, error) {
	net, err := p.Provider.GetNetwork(networkID, channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}
	v, err := net.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault for [%s:%s:%s]", networkID, channel, namespace)
	}
	return v, nil
}

type PublicParamsProvider struct {
	Provider *network.Provider
}

func (p *PublicParamsProvider) PublicParams(networkID string, channel string, namespace string) ([]byte, error) {
	net, err := p.Provider.GetNetwork(networkID, channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}
	v, err := net.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault for [%s:%s:%s]", networkID, channel, namespace)
	}
	return v.QueryEngine().PublicParams()
}
