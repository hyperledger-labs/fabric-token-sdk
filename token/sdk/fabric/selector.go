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

type GetFabricNetworkServiceFunc func(network string) fabric.NetworkService

type LockerProvider struct {
	sp      view.ServiceProvider
	timeout time.Duration
}

func NewLockerProvider(sp view.ServiceProvider, timeout time.Duration) *LockerProvider {
	return &LockerProvider{sp: sp, timeout: timeout}
}

func (s *LockerProvider) New(network string, channel string, namespace string) selector.Locker {
	ch, err := fabric.GetFabricNetworkService(s.sp, network).Channel(channel)
	if err != nil {
		panic(err)
	}
	return inmemory.NewLocker(ch, s.timeout)
}
