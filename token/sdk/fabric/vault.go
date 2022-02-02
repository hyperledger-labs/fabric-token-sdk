/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabric

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
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
	ch := fabric.GetChannel(v.sp, network, channel)
	res = vault.New(
		v.sp,
		ch.Name(),
		namespace,
		fabric3.NewVault(ch),
	)

	// update cache
	v.vaultCache[k] = res

	return res
}
