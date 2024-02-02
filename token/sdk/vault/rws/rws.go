/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rws

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor/rws"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	rws2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws"
	"github.com/pkg/errors"
)

type VaultProvider struct {
	sp view.ServiceProvider

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]vault.TokenVault
}

func NewVaultProvider(sp view.ServiceProvider) *VaultProvider {
	return &VaultProvider{sp: sp, vaultCache: make(map[string]vault.TokenVault)}
}

func (v *VaultProvider) Vault(network string, channel string, namespace string) (vault.TokenVault, error) {
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

	tokenStore, err := rws.NewTokenStore(v.sp, token.TMSID{
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
		tmsID := token.TMSID{
			Network:   network,
			Channel:   ch.Name(),
			Namespace: namespace,
		}
		storage, err := storageProvider.NewStorage(tmsID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new storage")
		}
		res, err = rws2.NewVault(storage, tmsID, fabric2.NewVault(ch, tokenStore))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new vault")
		}
	} else {
		ons := orion.GetOrionNetworkService(v.sp, network)
		if ons == nil {
			return nil, errors.Errorf("cannot find network [%s]", network)
		}
		tmsID := token.TMSID{
			Network:   network,
			Channel:   "",
			Namespace: namespace,
		}
		storage, err := storageProvider.NewStorage(tmsID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new storage")
		}
		res, err = rws2.NewVault(storage, tmsID, orion2.NewVault(ons, tokenStore))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create new vault")
		}
	}

	// update cache
	v.vaultCache[k] = res

	return res, nil
}
