/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package simple

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-sdk.selector.simple")

type LockerProvider interface {
	New(network, channel, namespace string) Locker
}

type SelectorService struct {
	managerLazyCache common.LazyProvider[*token.ManagementService, token.SelectorManager]
}

func NewProvider(lockerProvider LockerProvider, numRetry int, timeout time.Duration, tracer Tracer) *SelectorService {
	loader := &loader{
		lockerProvider:       lockerProvider,
		tracer:               tracer,
		numRetry:             numRetry,
		timeout:              timeout,
		requestCertification: true,
	}
	return &SelectorService{
		managerLazyCache: common.NewLazyProviderWithKeyMapper(key, loader.load),
	}
}

func (s *SelectorService) SelectorManager(tms *token.ManagementService) (token.SelectorManager, error) {
	if tms == nil {
		return nil, errors.Errorf("invalid tms, nil reference")
	}
	return s.managerLazyCache.Get(tms)
}

type Cache interface {
	Get(key string) (interface{}, bool)
	Add(key string, value interface{})
}

type queryService struct {
	qe     QueryService
	locker Locker
}

func (q *queryService) UnspentTokensIterator() (*token.UnspentTokensIterator, error) {
	return q.qe.UnspentTokensIterator()
}

func (q *queryService) UnspentTokensIteratorBy(id, typ string) (driver.UnspentTokensIterator, error) {
	return q.qe.UnspentTokensIteratorBy(id, typ)
}

func (q *queryService) GetTokens(inputs ...*token2.ID) ([]*token2.Token, error) {
	return q.qe.GetTokens(inputs...)
}

type loader struct {
	lockerProvider       LockerProvider
	tracer               Tracer
	numRetry             int
	timeout              time.Duration
	requestCertification bool
}

func (s *loader) load(tms *token.ManagementService) (token.SelectorManager, error) {
	logger.Debugf("new in-memory locker for [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())

	locker := s.lockerProvider.New(tms.Network(), tms.Channel(), tms.Namespace())
	qe := &queryService{
		qe:     tms.Vault().NewQueryEngine(),
		locker: locker,
	}
	pp := tms.PublicParametersManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public parameters not set yet for TMS [%s]", tms.ID())
	}

	return NewManager(
		locker,
		func() QueryService { return qe },
		s.numRetry,
		s.timeout,
		s.requestCertification,
		pp.Precision(),
		s.tracer,
	), nil
}

func key(tms *token.ManagementService) string {
	return tms.Network() + tms.Channel() + tms.Namespace()
}
