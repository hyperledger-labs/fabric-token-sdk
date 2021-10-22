/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
)

type VaultProvider struct {
	sp view.ServiceProvider
}

func NewVaultProvider(sp view.ServiceProvider) *VaultProvider {
	return &VaultProvider{sp: sp}
}

func (v *VaultProvider) Vault(network string, channel string, namespace string) driver.Vault {
	ch := fabric.GetChannel(v.sp, network, channel)
	return vault.New(
		v.sp,
		ch.Name(),
		namespace,
		fabric3.NewVault(ch),
	)
}
