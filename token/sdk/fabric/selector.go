/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabric

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/inmemory"
)

type Vault struct {
	*fabric.Vault
}

func (v *Vault) Status(id string) (int, error) {
	r, _, err := v.Vault.Status(id)
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
	ch, err := fabric.GetFabricNetworkService(s.sp, network).Channel(channel)
	if err != nil {
		panic(err)
	}
	return inmemory.NewLocker(&Vault{Vault: ch.Vault()}, s.sleepTimeout, s.validTxEvictionTimeoutMillis)
}
