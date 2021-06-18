/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
)

type VaultProvider struct {
	sp view.ServiceProvider
}

func NewVaultProvider(sp view.ServiceProvider) *VaultProvider {
	return &VaultProvider{sp: sp}
}

func (v *VaultProvider) Vault(network string, channel string, namespace string) driver.Vault {
	return vault.NewVault(
		v.sp,
		fabric.GetChannel(v.sp, network, channel),
		namespace,
	)
}
