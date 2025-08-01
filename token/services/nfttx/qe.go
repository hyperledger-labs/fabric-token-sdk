/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"context"
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/marshaller"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"github.com/thedevsaddam/gojsonq"
)

var (
	// ErrNoResults is returned when no results are found
	ErrNoResults = errors.New("no results found")
)

type vault interface {
	GetTokens(ctx context.Context, inputs ...*token2.ID) ([]*token2.Token, error)
}

type selector interface {
	Filter(filter Filter, q string) ([]*token2.ID, error)
}

type QueryExecutor struct {
	selector
	vault
	precision uint64
}

func NewQueryExecutor(sp token.ServiceProvider, wallet string, precision uint64, opts ...token.ServiceOption) (*QueryExecutor, error) {
	tms := token.GetManagementService(sp, opts...)
	qe := tms.Vault().NewQueryEngine()
	return &QueryExecutor{
		selector: NewFilter(
			wallet,
			qe,
			tms.PublicParametersManager().PublicParameters().Precision(),
		),
		vault:     qe,
		precision: precision,
	}, nil
}

func (s *QueryExecutor) QueryByKey(ctx context.Context, state interface{}, key string, value string) error {
	ids, err := s.Filter(&jsonFilter{
		q:     gojsonq.New(),
		key:   key,
		value: value,
	}, "1")
	if err != nil {
		if errors.Is(errors.Cause(err), ErrNoResults) {
			return ErrNoResults
		}
		return errors.Wrap(err, "failed to filter")
	}
	tokens, err := s.GetTokens(ctx, ids...)
	if err != nil {
		return errors.Wrap(err, "failed to get tokens")
	}
	for _, t := range tokens {
		q, err := token2.ToQuantity(t.Quantity, s.precision)
		if err != nil {
			return errors.Wrap(err, "failed to convert quantity")
		}
		if q.Cmp(token2.NewOneQuantity(s.precision)) == 0 {
			// this is the token
			decoded, err := base64.StdEncoding.DecodeString(string(t.Type))
			if err != nil {
				return errors.Wrap(err, "failed to decode type")
			}
			if err := marshaller.Unmarshal(decoded, state); err == nil {
				return errors.Wrap(err, "failed to unmarshal state")
			}
		}
	}
	return ErrNoResults
}

type jsonFilter struct {
	q          *gojsonq.JSONQ
	key, value string
}

func (j *jsonFilter) ContainsToken(token *token2.UnspentToken) bool {
	decoded, err := base64.StdEncoding.DecodeString(string(token.Type))
	if err != nil {
		logger.Debugf("failed to decode token type [%s]", token.Type)
		return false
	}
	logger.Debugf("decoded token type [%s]", string(decoded))
	jq := j.q.FromString(string(decoded))
	res := jq.Find(j.key)
	if v, ok := res.(string); ok {
		return v == j.value
	}
	logger.Debugf("res [%s] for [%s,%s]", res, j.key, j.value)
	return false
}
