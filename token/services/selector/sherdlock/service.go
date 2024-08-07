/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

type SelectorService struct {
	managerLazyCache utils.LazyProvider[*token.ManagementService, token.SelectorManager]
}

func NewService(tokenDBManager *tokendb.Manager, tokenLockDBManager *tokenlockdb.Manager, cfg driver.ConfigService, metricsProvider metrics.Provider) *SelectorService {
	c := SelectorConfig{}
	err := cfg.UnmarshalKey("token.selector", &c)
	if err != nil {
		panic("invalid config for key [token.selector]: expect retryInterval (duration) and numRetries (integer))")
	}

	loader := &loader{
		tokenDBManager:     tokenDBManager,
		tokenLockDBManager: tokenLockDBManager,
		m:                  newMetrics(metricsProvider),
		retryInterval:      c.GetRetryInterval(),
		numRetries:         c.GetNumRetries(),
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
	tokenDBManager     *tokendb.Manager
	tokenLockDBManager *tokenlockdb.Manager
	m                  *Metrics
	numRetries         int
	retryInterval      time.Duration
}

func (s *loader) load(tms *token.ManagementService) (token.SelectorManager, error) {
	pp := tms.PublicParametersManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public parameters not set yet for TMS [%s]", tms.ID())
	}
	tokenDB, err := s.tokenDBManager.DBByTMSId(tms.ID())
	if err != nil {
		return nil, errors.Errorf("failed to create tokenDB: %v", err)
	}
	tokenLockDB, err := s.tokenLockDBManager.DBByTMSId(tms.ID())
	if err != nil {
		return nil, errors.Errorf("failed to create tokenLockDB: %v", err)
	}

	return NewManager(tokenDB, tokenLockDB, s.m, pp.Precision(), s.retryInterval, s.numRetries), nil
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
