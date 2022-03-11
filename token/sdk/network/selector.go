/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"fmt"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/inmemory"
	"time"
)

type FabricVault struct {
	*fabric.Vault
}

func (v *FabricVault) Status(id string) (int, error) {
	r, _, err := v.Vault.Status(id)
	return int(r), err
}

type OrionVault struct {
	*orion.Vault
}

func (v *OrionVault) Status(id string) (int, error) {
	r, err := v.Vault.Status(id)
	return int(r), err
}

type LockerProvider struct {
	sp                           view.ServiceProvider
	sleepTimeout                 time.Duration
	validTxEvictionTimeoutMillis int64
}

func NewLockerProvider(sp view.ServiceProvider, sleepTimeout time.Duration, validTxEvictionTimeoutMillis int64) *LockerProvider {
	return &LockerProvider{sp: sp, sleepTimeout: sleepTimeout, validTxEvictionTimeoutMillis: validTxEvictionTimeoutMillis}
}

func (s *LockerProvider) New(network string, channel string, namespace string) selector.Locker {
	fns := fabric.GetFabricNetworkService(s.sp, network)
	if fns != nil {
		ch, err := fns.Channel(channel)
		if err == nil {
			return inmemory.NewLocker(&FabricVault{Vault: ch.Vault()}, s.sleepTimeout, s.validTxEvictionTimeoutMillis)
		}
	}
	ons := orion.GetOrionNetworkService(s.sp, network)
	if ons == nil {
		panic(fmt.Sprintf("network %s not found", network))
	}
	return inmemory.NewLocker(&OrionVault{Vault: ons.Vault()}, s.sleepTimeout, s.validTxEvictionTimeoutMillis)
}
