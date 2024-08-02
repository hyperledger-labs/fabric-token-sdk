/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

const (
	retrySelectionBackoff = 5 * time.Second
	cleanupTickPeriod     = 1 * time.Minute
	cleanupPeriod         = 30 * time.Second
)

type SelectorService struct {
	managerLazyCache utils.LazyProvider[*token.ManagementService, token.SelectorManager]
}

func NewService(tokenDBManager *tokendb.Manager, tokenLockDBManager *tokenlockdb.Manager, metricsProvider metrics.Provider) *SelectorService {
	loader := &loader{
		tokenDBManager:     tokenDBManager,
		tokenLockDBManager: tokenLockDBManager,
		m:                  newMetrics(metricsProvider),
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
	return NewManager(
		tokenDB,
		tokenLockDB,
		s.m,
		pp.Precision(),
		retrySelectionBackoff,
		cleanupTickPeriod,
	), nil
}

func key(tms *token.ManagementService) string {
	return tms.ID().String()
}
