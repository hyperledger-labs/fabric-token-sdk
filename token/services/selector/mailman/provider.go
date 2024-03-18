/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

type Subscribe = events.Subscriber

type VaultProvider interface {
	Vault(tms *token.ManagementService) (Vault, QueryService, error)
}

type SelectorService struct {
	vaultProvider VaultProvider
	subscribe     Subscribe
	tracer        Tracer

	lock     sync.RWMutex
	managers map[string]token.SelectorManager
	// TODO create a shared worker pool for all selectors
	// workerPool []*worker
}

func NewService(vaultProvider VaultProvider, subscribe Subscribe, tracer Tracer) *SelectorService {
	return &SelectorService{
		vaultProvider: vaultProvider,
		subscribe:     subscribe,
		tracer:        tracer,
		managers:      make(map[string]token.SelectorManager),
	}
}

func (s *SelectorService) SelectorManager(tms *token.ManagementService) (token.SelectorManager, error) {
	if tms == nil {
		return nil, errors.Errorf("invalid tms, nil reference")
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

	// create walletID extractor function using TMS wallet manager
	walletIDByRawIdentity := func(rawIdentity []byte) string {
		w := tms.WalletManager().OwnerWallet(rawIdentity)
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
	vault, qs, err := s.vaultProvider.Vault(tms)
	if err != nil {
		return nil, errors.Errorf("cannot get ntwork vault for TMS [%s]", tms.ID())
	}
	newManager, err := NewManager(
		tms.ID(),
		vault,
		qs,
		walletIDByRawIdentity,
		s.tracer,
		pp.Precision(),
		s.subscribe,
	)
	if err != nil {
		return nil, err
	}
	newManager.Start()
	s.managers[key] = newManager
	return newManager, nil
}
