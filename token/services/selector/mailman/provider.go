/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

type Subscribe = events.Subscriber

type VaultProvider interface {
	Vault(tms *token.ManagementService) (QueryService, error)
}

type SelectorService struct {
	managerLazyCache utils.LazyProvider[*token.ManagementService, token.SelectorManager]
	// TODO create a shared worker pool for all selectors
	// workerPool []*worker
}

func NewService(subscribe Subscribe, tracer Tracer) *SelectorService {
	loader := &loader{
		subscribe: subscribe,
		tracer:    tracer,
	}
	return &SelectorService{
		managerLazyCache: utils.NewLazyProviderWithKeyMapper(key, loader.load),
	}
}

func (s *SelectorService) SelectorManager(tms *token.ManagementService) (token.SelectorManager, error) {
	if tms == nil {
		return nil, errors.Errorf("invalid tms, nil reference")
	}

	return s.managerLazyCache.Get(tms)
}

type loader struct {
	subscribe Subscribe
	tracer    Tracer
}

func (s *loader) load(tms *token.ManagementService) (token.SelectorManager, error) {
	walletIDByRawIdentity := func(rawIdentity []byte) string {
		w := tms.WalletManager().OwnerWallet(rawIdentity)
		if w == nil {
			logger.Errorf("wallet not found for identity [%s][%s]", token.Identity(rawIdentity), debug.Stack())
			return ""
		}
		return w.ID()
	}

	pp := tms.PublicParametersManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public parameters not set yet for TMS [%s]", tms.ID())
	}
	newManager, err := NewManager(
		tms.ID(),
		tms.Vault().NewQueryEngine(),
		walletIDByRawIdentity,
		s.tracer,
		pp.Precision(),
		s.subscribe,
	)
	if err != nil {
		return nil, err
	}
	newManager.Start()
	return newManager, nil
}

func key(tms *token.ManagementService) string {
	return tms.Network() + tms.Channel() + tms.Namespace()
}
