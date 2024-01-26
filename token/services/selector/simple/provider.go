/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package simple

import (
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.selector")

type Transaction interface {
	ID() string
	Network() string
	Channel() string
	Namespace() string
}

type Repository interface {
	Next(typ string, of token.OwnerFilter) (*token2.UnspentToken, error)
}

type LockerProvider interface {
	New(network, channel, namespace string) Locker
}

type SelectorService struct {
	tracer               Tracer
	numRetry             int
	timeout              time.Duration
	requestCertification bool

	lock           sync.RWMutex
	lockerProvider LockerProvider
	lockers        map[string]Locker
	managers       map[string]token.SelectorManager
}

func NewProvider(lockerProvider LockerProvider, numRetry int, timeout time.Duration, tracer Tracer) *SelectorService {
	return &SelectorService{
		lockerProvider:       lockerProvider,
		lockers:              map[string]Locker{},
		managers:             map[string]token.SelectorManager{},
		numRetry:             numRetry,
		timeout:              timeout,
		requestCertification: true,
		tracer:               tracer,
	}
}

func (s *SelectorService) SelectorManager(tms *token.ManagementService) (token.SelectorManager, error) {
	if tms == nil {
		return nil, errors.Errorf("invalid tms, nil reference")
	}

	key := tms.Network() + tms.Channel() + tms.Namespace()

	s.lock.RLock()
	manager, ok := s.managers[key]
	if ok {
		s.lock.RUnlock()
		return manager, nil
	}
	s.lock.RUnlock()

	s.lock.RLock()
	defer s.lock.RUnlock()

	manager, ok = s.managers[key]
	if ok {
		return manager, nil
	}

	// instantiate a new manager
	locker, ok := s.lockers[key]
	if !ok {
		logger.Debugf("new in-memory locker for [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())
		locker = s.lockerProvider.New(tms.Network(), tms.Channel(), tms.Namespace())
		s.lockers[key] = locker
	} else {
		logger.Debugf("in-memory selector for [%s:%s:%s] exists", tms.Network(), tms.Channel(), tms.Namespace())
	}
	qe := newQueryService(
		tms.Vault().NewQueryEngine(),
		locker,
	)
	pp := tms.PublicParametersManager().PublicParameters()
	if pp == nil {
		return nil, errors.Errorf("public parameters not set yet for TMS [%s]", tms.ID())
	}
	manager = NewManager(
		locker,
		func() QueryService {
			return qe
		},
		s.numRetry,
		s.timeout,
		s.requestCertification,
		pp.Precision(),
		s.tracer,
	)
	s.managers[key] = manager
	return manager, nil
}

func (s *SelectorService) SetNumRetries(n uint) {
	s.numRetry = int(n)
}

func (s *SelectorService) SetRetryTimeout(t time.Duration) {
	s.timeout = t
}

func (s *SelectorService) SetRequestCertification(v bool) {
	s.requestCertification = v
}

type Cache interface {
	Get(key string) (interface{}, bool)
	Add(key string, value interface{})
}

type queryService struct {
	qe     QueryService
	locker Locker
}

func newQueryService(qe QueryService, locker Locker) *queryService {
	return &queryService{
		qe:     qe,
		locker: locker,
	}
}

func (q *queryService) UnspentTokensIterator() (*token.UnspentTokensIterator, error) {
	return q.qe.UnspentTokensIterator()
}

func (q *queryService) UnspentTokensIteratorBy(id, typ string) (*token.UnspentTokensIterator, error) {
	return q.qe.UnspentTokensIteratorBy(id, typ)
}

func (q *queryService) GetTokens(inputs ...*token2.ID) ([]*token2.Token, error) {
	return q.qe.GetTokens(inputs...)
}
