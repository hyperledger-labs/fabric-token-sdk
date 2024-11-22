/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	lazy2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/pkg/errors"
)

type SelectorService struct {
	managerLazyCache lazy2.Provider[*token.ManagementService, token.SelectorManager]
}

func NewService(fetcherProvider FetcherProvider, tokenLockDBManager *tokenlockdb.Manager, c core.ConfigProvider) *SelectorService {
	cfg, err := config.New(c)
	if err != nil {
		logger.Errorf("error getting selector config, using defaults. %s", err.Error())
	}

	loader := &loader{
		tokenLockDBManager:     tokenLockDBManager,
		fetcherProvider:        fetcherProvider,
		retryInterval:          cfg.GetRetryInterval(),
		numRetries:             cfg.GetNumRetries(),
		leaseExpiry:            cfg.GetLeaseExpiry(),
		leaseCleanupTickPeriod: cfg.GetLeaseCleanupTickPeriod(),
	}
	return &SelectorService{
		managerLazyCache: lazy2.NewProviderWithKeyMapper(key, loader.load),
	}
}

func (s *SelectorService) SelectorManager(tms *token.ManagementService) (token.SelectorManager, error) {
	if tms == nil {
		return nil, errors.Errorf("invalid tms, nil reference")
	}

	return s.managerLazyCache.Get(tms)
}

type loader struct {
	tokenLockDBManager     *tokenlockdb.Manager
	fetcherProvider        FetcherProvider
	numRetries             int
	retryInterval          time.Duration
	leaseExpiry            time.Duration
	leaseCleanupTickPeriod time.Duration
}

func (s *loader) load(tms *token.ManagementService) (token.SelectorManager, error) {
	pp := tms.PublicParametersManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public parameters not set yet for TMS [%s]", tms.ID())
	}
	tokenLockDB, err := s.tokenLockDBManager.DBByTMSId(tms.ID())
	if err != nil {
		return nil, errors.Errorf("failed to create tokenLockDB: %v", err)
	}
	fetcher, err := s.fetcherProvider.GetFetcher(tms.ID())
	if err != nil {
		return nil, errors.Errorf("failed to create token fetcher: %v", err)
	}
	return NewManager(
		fetcher,
		tokenLockDB,
		pp.Precision(),
		s.retryInterval,
		s.numRetries,
		s.leaseExpiry,
		s.leaseCleanupTickPeriod,
	), nil
}

func key(tms *token.ManagementService) string {
	return tms.ID().String()
}
