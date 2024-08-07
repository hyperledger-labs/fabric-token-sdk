/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	lazy2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/pkg/errors"
)

const (
	cleanupTickPeriod = 1 * time.Minute
	cleanupPeriod     = 30 * time.Second
)

type SelectorService struct {
	managerLazyCache lazy2.Provider[*token.ManagementService, token.SelectorManager]
}

func NewService(fetcherProvider FetcherProvider, tokenLockDBManager *tokenlockdb.Manager, cfg driver.ConfigService) *SelectorService {
	c := SelectorConfig{}
	err := cfg.UnmarshalKey("token.selector", &c)
	if err != nil {
		panic("invalid config for key [token.selector]: expect retryInterval (duration) and numRetries (integer))")
	}

	loader := &loader{
		tokenLockDBManager: tokenLockDBManager,
		fetcherProvider:    fetcherProvider,
		retryInterval:      c.GetRetryInterval(),
		numRetries:         c.GetNumRetries(),
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
	tokenLockDBManager *tokenlockdb.Manager
	fetcherProvider    FetcherProvider
	numRetries         int
	retryInterval      time.Duration
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
		cleanupTickPeriod,
	), nil
}

func key(tms *token.ManagementService) string {
	return tms.ID().String()
}

type SelectorConfig struct {
	RetryInterval time.Duration
	NumRetries    int
}

func (c *SelectorConfig) GetNumRetries() int {
	if c.NumRetries > 0 {
		return c.NumRetries
	} else {
		return 3
	}
}
func (c *SelectorConfig) GetRetryInterval() time.Duration {
	if c.RetryInterval > 0 {
		return c.RetryInterval
	} else {
		return 5 * time.Second
	}
}
