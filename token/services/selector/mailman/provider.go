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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

type SelectorService struct {
	sp       view.ServiceProvider
	lock     sync.RWMutex
	managers map[string]token.SelectorManager
	// TODO create a shared worker pool for all selectors
	// workerPool []*worker
}

func NewService(sp view.ServiceProvider) *SelectorService {
	return &SelectorService{
		sp:       sp,
		managers: make(map[string]token.SelectorManager),
	}
}

func (s *SelectorService) SelectorManager(networkID string, channel string, namespace string) (token.SelectorManager, error) {
	tms := token.GetManagementService(
		s.sp,
		token.WithNetwork(networkID),
		token.WithChannel(channel),
		token.WithNamespace(namespace),
	)
	if tms == nil {
		return nil, errors.Errorf("failed to get TMS for [%s:%s:%s]", networkID, channel, namespace)
	}

	key := tms.Network() + tms.Channel() + tms.Namespace()

	// if Manager for this network/channel/namespace already exists, just return it
	s.lock.RLock()
	m, ok := s.managers[key]
	if ok {
		s.lock.RUnlock()
		return m, nil
	}
	s.lock.RUnlock()

	s.lock.Lock()
	defer s.lock.Unlock()

	// check again if Manager for this network/channel/namespace already exists, just return it
	m, ok = s.managers[key]
	if ok {
		return m, nil
	}

	// otherwise, build a new Manager

	// get notification service
	sub, err := events.GetSubscriber(s.sp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get event subscriber")
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

	pp := tms.PublicParametersManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public parameters not set yet for TMS [%s]", tms.ID())
	}
	vault, err := network.GetInstance(s.sp, tms.Network(), tms.Channel()).Vault(tms.Namespace())
	if err != nil {
		return nil, errors.Errorf("cannot get ntwork vault for TMS [%s]", tms.ID())
	}
	newManager := NewManager(tms.ID(), vault, tms.Vault().NewQueryEngine(), walletIDByRawIdentity, tracing.Get(s.sp).GetTracer(), pp.Precision(), sub)
	newManager.Start()
	s.managers[key] = newManager
	return newManager, nil
}
