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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
)

type LockerProvider struct {
	ttxStoreServiceManager db.StoreServiceManager[*ttxdb.StoreService]
	sleepTimeout           time.Duration
	validTxEvictionTimeout time.Duration
}

func NewLockerProvider(
	ttxStoreServiceManager db.StoreServiceManager[*ttxdb.StoreService],
	sleepTimeout time.Duration,
	validTxEvictionTimeout time.Duration,
) *LockerProvider {
	return &LockerProvider{
		ttxStoreServiceManager: ttxStoreServiceManager,
		sleepTimeout:           sleepTimeout,
		validTxEvictionTimeout: validTxEvictionTimeout,
	}
}

func (s *LockerProvider) New(network, channel, namespace string) (selector.Locker, error) {
	db, err := s.ttxStoreServiceManager.StoreServiceByTMSId(token.TMSID{
		Network:   network,
		Channel:   channel,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	return inmemory.NewLocker(db, s.sleepTimeout, s.validTxEvictionTimeout), nil
}
