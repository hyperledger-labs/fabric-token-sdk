/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package selector

import (
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
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

type selectorService struct {
	sp                   view.ServiceProvider
	numRetry             int
	timeout              time.Duration
	requestCertification bool

	lock           sync.Mutex
	lockerProvider LockerProvider
	lockers        map[string]Locker
	managers       map[string]token.SelectorManager
}

func NewProvider(sp view.ServiceProvider, lockerProvider LockerProvider, numRetry int, timeout time.Duration) *selectorService {
	return &selectorService{
		sp:                   sp,
		lockerProvider:       lockerProvider,
		lockers:              map[string]Locker{},
		managers:             map[string]token.SelectorManager{},
		numRetry:             numRetry,
		timeout:              timeout,
		requestCertification: true,
	}
}

func (s *selectorService) SelectorManager(network string, channel string, namespace string) token.SelectorManager {
	tms := token.GetManagementService(
		s.sp,
		token.WithNetwork(network),
		token.WithChannel(channel),
		token.WithNamespace(namespace),
	)

	s.lock.Lock()
	defer s.lock.Unlock()

	key := tms.Network() + tms.Channel() + tms.Namespace()
	manager, ok := s.managers[key]
	if ok {
		return manager
	}

	// instantiate a new manager
	locker, ok := s.lockers[key]
	if !ok {
		logger.Debugf("new in-memory locker for [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())
		locker = s.lockerProvider.New(network, channel, namespace)
		s.lockers[key] = locker
	} else {
		logger.Debugf("in-memory selector for [%s:%s:%s] exists", tms.Network(), tms.Channel(), tms.Namespace())
	}
	qe := newQueryService(
		tms.Vault().NewQueryEngine(),
		locker,
	)
	manager = NewManager(
		locker,
		func() QueryService {
			return qe
		},
		s.numRetry,
		s.timeout,
		s.requestCertification,
		tms.PublicParametersManager().Precision(),
		tracing.Get(s.sp).GetTracer(),
	)
	s.managers[key] = manager
	return manager
}

func (s *selectorService) SetNumRetries(n uint) {
	s.numRetry = int(n)
}

func (s *selectorService) SetRetryTimeout(t time.Duration) {
	s.timeout = t
}

func (s *selectorService) SetRequestCertification(v bool) {
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
