/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"sync"
	"sync/atomic"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services"
)

type VersionKeeperProvider lazy.Provider[token.TMSID, *VersionKeeper]

func NewVersionKeeperProvider() VersionKeeperProvider {
	return lazy.NewProviderWithKeyMapper(services.Key, func(token.TMSID) (*VersionKeeper, error) {
		return &VersionKeeper{}, nil
	})
}

type VersionKeeper struct {
	version atomic.Uint64
	once    sync.Once
}

func (k *VersionKeeper) GetVersion() uint64 {
	return k.version.Load()
}

func (k *VersionKeeper) UpdateVersion() {
	var init bool
	k.once.Do(func() {
		// note that in the case when we call update the very first time ...
		// we expect this to be an initialize call
		init = true
	})
	if init {
		return
	}

	v := k.version.Add(1)
	logger.Infof("Updated PP version to %d", v)
}
