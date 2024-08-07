/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package simple

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	viewdriver "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-sdk.selector.simple")

type LockerProvider interface {
	New(network, channel, namespace string) (Locker, error)
}

type SelectorService struct {
	managerLazyCache lazy.Provider[*token.ManagementService, token.SelectorManager]
}

func NewService(tms *token.ManagementService, lockerProvider LockerProvider, cfg viewdriver.ConfigService) *SelectorService {
	c := SelectorConfig{}
	err := cfg.UnmarshalKey("token.selector", &c)
	if err != nil {
		panic("invalid config for key [token.selector]: expect retryInterval (duration) and numRetries (integer))")
	}

	loader := &loader{
		lockerProvider:       lockerProvider,
		numRetries:           c.GetNumRetries(),
		retryInterval:        c.GetRetryInterval(),
		requestCertification: true,
	}
	return &SelectorService{
		managerLazyCache: lazy.NewProviderWithKeyMapper(key, loader.load),
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

func (q *queryService) UnspentTokensIteratorBy(ctx context.Context, id, tokenType string) (driver.UnspentTokensIterator, error) {
	return q.qe.UnspentTokensIteratorBy(ctx, id, tokenType)
}

func (q *queryService) GetTokens(inputs ...*token2.ID) ([]*token2.Token, error) {
	return q.qe.GetTokens(inputs...)
}

type loader struct {
	lockerProvider       LockerProvider
	numRetries           int
	retryInterval        time.Duration
	requestCertification bool
}

func (s *loader) load(tms *token.ManagementService) (token.SelectorManager, error) {
	logger.Debugf("new in-memory locker for [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())

	locker, err := s.lockerProvider.New(tms.Network(), tms.Channel(), tms.Namespace())
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting locker")
	}
	qe := &queryService{
		qe:     tms.Vault().NewQueryEngine(),
		locker: locker,
	}

	return NewManager(
		locker,
		func() QueryService { return qe },
		s.numRetries,
		s.retryInterval,
		s.requestCertification,
		tms.PublicParametersManager().PublicParameters().Precision(),
	), nil
}

func key(tms *token.ManagementService) string {
	return tms.Network() + tms.Channel() + tms.Namespace()
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
	if c.RetryInterval != 0 {
		return c.RetryInterval
	} else {
		return 5 * time.Second
	}
}
