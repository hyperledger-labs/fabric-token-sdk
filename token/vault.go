/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// QueryEngine models a token query engine
type QueryEngine struct {
	qe driver.QueryEngine

	// Variable used to control retry condition
	NumRetries int
	RetryDelay time.Duration
}

func NewQueryEngine(qe driver.QueryEngine, numRetries int, retryDelay time.Duration) *QueryEngine {
	return &QueryEngine{qe: qe, NumRetries: numRetries, RetryDelay: retryDelay}
}

// IsMine returns true is the given token is in this vault and therefore owned by this client
func (q *QueryEngine) IsMine(id *token2.ID) (bool, error) {
	return q.qe.IsMine(id)
}

// UnspentTokensIterator returns an iterator over all unspent tokens stored in the vault
func (q *QueryEngine) UnspentTokensIterator() (*UnspentTokensIterator, error) {
	it, err := q.qe.UnspentTokensIterator()
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{UnspentTokensIterator: it}, nil
}

// UnspentTokensIteratorBy is an iterator over all unspent tokens in this vault owned by passed wallet id and whose token type matches the passed token type
func (q *QueryEngine) UnspentTokensIteratorBy(walletID, tokenType string) (*UnspentTokensIterator, error) {
	it, err := q.qe.UnspentTokensIteratorBy(walletID, tokenType)
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{UnspentTokensIterator: it}, nil
}

// ListUnspentTokens returns a list of all unspent tokens stored in the vault
func (q *QueryEngine) ListUnspentTokens() (*token2.UnspentTokens, error) {
	return q.qe.ListUnspentTokens()
}

func (q *QueryEngine) ListAuditTokens(ids ...*token2.ID) ([]*token2.Token, error) {
	var tokens []*token2.Token
	var err error

	for i := 0; i < q.NumRetries; i++ {
		tokens, err = q.qe.ListAuditTokens(ids...)

		if err != nil {
			// check if there is any token id whose corresponding transaction is pending
			// if there is, then wait a bit and retry to load the outputs
			retry := false
			for _, id := range ids {
				pending, err := q.qe.IsPending(id)
				if err != nil {
					break
				}
				if pending {
					logger.Warnf("cannot get audit token for id [%d] because the relative transaction is pending, retry at [%d]", id, i)
					if i == q.NumRetries-1 {
						return nil, errors.Wrapf(err, "failed to get audit tokens, tx [%s] is still pending", id.TxId)
					}
					time.Sleep(q.RetryDelay)
					retry = true
					break
				}
			}

			if retry {
				tokens = nil
				continue
			}

			return nil, errors.Wrapf(err, "failed to get audit tokens")
		}

		// The execution was successful, we can stop
		break
	}

	return tokens, nil
}

func (q *QueryEngine) ListHistoryIssuedTokens() (*token2.IssuedTokens, error) {
	return q.qe.ListHistoryIssuedTokens()
}

// PublicParams returns the public parameters stored in the vault
func (q *QueryEngine) PublicParams() ([]byte, error) {
	return q.qe.PublicParams()
}

// GetTokens returns the tokens stored in the vault matching the given ids
func (q *QueryEngine) GetTokens(inputs ...*token2.ID) ([]*token2.Token, error) {
	_, tokens, err := q.qe.GetTokens(inputs...)
	return tokens, err
}

// Vault models a token vault
type Vault struct {
	v driver.Vault
}

// NewQueryEngine returns a new query engine
func (v *Vault) NewQueryEngine() *QueryEngine {
	return NewQueryEngine(v.v.QueryEngine(), 3, 3*time.Second)
}

// UnspentTokensIterator models an iterator over all unspent tokens stored in the vault
type UnspentTokensIterator struct {
	driver.UnspentTokensIterator
}

// Sum  computes the sum of the quantities of the tokens in the iterator.
// Sum closes the iterator at the end of the execution.
func (u *UnspentTokensIterator) Sum(precision uint64) (token2.Quantity, error) {
	defer u.Close()
	sum := token2.NewZeroQuantity(precision)
	for {
		tok, err := u.Next()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			break
		}

		q, err := token2.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		sum = sum.Add(q)
	}

	return sum, nil
}
