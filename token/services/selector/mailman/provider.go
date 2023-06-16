/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type selectorService struct {
	sp       view.ServiceProvider
	lock     sync.Mutex
	managers map[string]token.SelectorManager
	// TODO create a shared worker pool for all selectors
	// workerPool []*worker
}

func NewService(sp view.ServiceProvider) *selectorService {
	return &selectorService{
		sp:       sp,
		lock:     sync.Mutex{},
		managers: make(map[string]token.SelectorManager),
	}
}

func (s *selectorService) SelectorManager(network string, channel string, namespace string) token.SelectorManager {
	tms := token.GetManagementService(
		s.sp,
		token.WithNetwork(network),
		token.WithChannel(channel),
		token.WithNamespace(namespace),
	)
	// TODO do something if we cannot get tms

	key := tms.Network() + tms.Channel() + tms.Namespace()

	s.lock.Lock()
	defer s.lock.Unlock()

	m, ok := s.managers[key]
	if ok {
		// if Manager for this network/channel/namespace already exists, just return it
		return m
	}

	// otherwise, build a new Manager

	// get notification service
	sub, err := events.GetSubscriber(s.sp)
	if err != nil {
		// TODO is this the proper error handling for this situation?
		return nil
	}

	// create walletID extractor function using TMS wallet manager
	walletIDByRawIdentity := func(rawIdentity []byte) string {
		w := tms.WalletManager().OwnerWalletByIdentity(rawIdentity)
		if w == nil {
			logger.Errorf("wallet not found for identity [%s][%s]", view2.Identity(rawIdentity), debug.Stack())
			return ""
		}
		return w.ID()
	}

	m = NewManager(tms.Vault().NewQueryEngine(), walletIDByRawIdentity, tracing.Get(s.sp).GetTracer(), tms.PublicParametersManager().PublicParameters().Precision(), sub)

	s.managers[key] = m
	return m
}
