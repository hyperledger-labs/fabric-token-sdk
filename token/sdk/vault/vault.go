/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package vault

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
)

type VaultProvider struct {
	sp view.ServiceProvider

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.Vault
}

func NewVaultProvider(sp view.ServiceProvider) *VaultProvider {
	return &VaultProvider{sp: sp, vaultCache: make(map[string]driver.Vault)}
}

func (v *VaultProvider) Vault(network string, channel string, namespace string) driver.Vault {
	k := network + channel + namespace
	// Check cache
	v.vaultCacheLock.RLock()
	res, ok := v.vaultCache[k]
	v.vaultCacheLock.RUnlock()
	if ok {
		return res
	}

	// lock
	v.vaultCacheLock.Lock()
	defer v.vaultCacheLock.Unlock()

	// check cache again
	res, ok = v.vaultCache[k]
	if ok {
		return res
	}

	// Create new vault
	fns := fabric.GetFabricNetworkService(v.sp, network)
	if fns != nil {
		ch := fabric.GetChannel(v.sp, network, channel)
		res = vault.New(
			v.sp,
			ch.Name(),
			namespace,
			fabric2.NewVault(ch),
		)
	} else {
		ons := orion.GetOrionNetworkService(v.sp, network)
		if ons != nil {
			res = vault.New(
				v.sp,
				"",
				namespace,
				orion2.NewVault(ons),
			)
		}
	}

	// update cache
	v.vaultCache[k] = res

	return res
}
