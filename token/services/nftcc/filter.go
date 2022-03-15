/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Filter is a filter for NFTCC
type Filter interface {
	// ContainsToken returns true if the passed token is recognized, false otherwise.
	ContainsToken(token *token2.UnspentToken) bool
}

type QueryService interface {
	UnspentTokensIterator() (*token.UnspentTokensIterator, error)
	GetTokens(inputs ...*token2.ID) ([]*token2.Token, error)
}

type MetricsAgent interface {
	EmitKey(val float32, event ...string)
}

type filter struct {
	queryService QueryService
	precision    uint64
	metricsAgent MetricsAgent
}

func NewFilter(service QueryService, metricsAgent MetricsAgent) *filter {
	return &filter{
		queryService: service,
		precision:    keys.Precision,
		metricsAgent: metricsAgent,
	}
}

func (s *filter) Filter(filter Filter, q string) ([]*token2.ID, error) {
	if filter == nil {
		return nil, errors.New("filter is nil")
	}
	ids, _, err := s.selectByFilter(filter, q)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to select tokens")
	}
	return ids, nil
}

func (s *filter) selectByFilter(filter Filter, q string) ([]*token2.ID, token2.Quantity, error) {
	var toBeSpent []*token2.ID
	var sum token2.Quantity
	target, err := token2.ToQuantity(q, s.precision)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to convert quantity")
	}

	unspentTokens, err := s.queryService.UnspentTokensIterator()
	if err != nil {
		return nil, nil, errors.Wrap(err, "token selection failed")
	}
	defer unspentTokens.Close()
	logger.Debugf("select token for a quantity of [%s]", q)

	sum = token2.NewZeroQuantity(s.precision)
	toBeSpent = nil
	for {
		t, err := unspentTokens.Next()
		if err != nil {
			return nil, nil, errors.Wrap(err, "token selection failed")
		}
		if t == nil {
			break
		}

		q := t.Quantity
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to convert quantity")
		}

		// check type and ownership
		selected := filter.ContainsToken(t)
		if !selected {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("token [%s,%s,%v] owner does not belong to the passed wallet", view.Identity(t.Owner.Raw), q, selected)
			}
			continue
		}

		// Append token
		logger.Debugf("adding quantity [%s]", q.Decimal())
		toBeSpent = append(toBeSpent, t.Id)
		sum = sum.Add(q)
		if target.Cmp(sum) <= 0 {
			return toBeSpent, sum, nil
		}
	}

	return nil, nil, errors.New("no token found")
}
