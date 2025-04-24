/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple/inmemory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
)

type TTXDBProvider interface {
	ServiceByTMSId(id token.TMSID) (*ttxdb.StoreService, error)
}

type LockerProvider struct {
	ttxdbProvider          TTXDBProvider
	sleepTimeout           time.Duration
	validTxEvictionTimeout time.Duration
}

func NewLockerProvider(ttxdbProvider TTXDBProvider, sleepTimeout time.Duration, validTxEvictionTimeout time.Duration) *LockerProvider {
	return &LockerProvider{
		ttxdbProvider:          ttxdbProvider,
		sleepTimeout:           sleepTimeout,
		validTxEvictionTimeout: validTxEvictionTimeout,
	}
}

func (s *LockerProvider) New(network, channel, namespace string) (selector.Locker, error) {
	db, err := s.ttxdbProvider.ServiceByTMSId(token.TMSID{
		Network:   network,
		Channel:   channel,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}
	return inmemory.NewLocker(db, s.sleepTimeout, s.validTxEvictionTimeout), nil
}
