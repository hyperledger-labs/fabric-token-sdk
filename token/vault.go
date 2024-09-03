/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// TxStatus is the status of a transaction
type TxStatus = driver.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown = driver.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = driver.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = driver.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = driver.Deleted
)

// QueryEngine models a token query engine
type QueryEngine struct {
	qe     driver.QueryEngine
	logger logging.Logger

	// Variables used to control retry condition
	NumRetries int
	RetryDelay time.Duration
}

func NewQueryEngine(logger logging.Logger, qe driver.QueryEngine, numRetries int, retryDelay time.Duration) *QueryEngine {
	return &QueryEngine{logger: logger, qe: qe, NumRetries: numRetries, RetryDelay: retryDelay}
}

// IsMine returns true is the given token is in this vault and therefore owned by this client
func (q *QueryEngine) IsMine(id *token.ID) (bool, error) {
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
func (q *QueryEngine) UnspentTokensIteratorBy(ctx context.Context, id, tokenType string) (driver.UnspentTokensIterator, error) {
	it, err := q.qe.UnspentTokensIteratorBy(ctx, id, tokenType)
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{UnspentTokensIterator: it}, nil
}

// ListUnspentTokens returns a list of all unspent tokens stored in the vault
func (q *QueryEngine) ListUnspentTokens() (*token.UnspentTokens, error) {
	return q.qe.ListUnspentTokens()
}

func (q *QueryEngine) ListAuditTokens(ids ...*token.ID) ([]*token.Token, error) {
	var tokens []*token.Token
	var err error

	for i := 0; i < q.NumRetries; i++ {
		tokens, err = q.qe.ListAuditTokens(ids...)

		if err != nil {
			// check if there is any token id whose corresponding transaction is pending
			// if there is, then wait a bit and retry to load the outputs
			retry := false
			for _, id := range ids {
				pending, err := q.qe.IsPending(id)
				if pending || err != nil {
					q.logger.Warnf("cannot get audit token for id [%s] because the relative transaction is pending, retry at [%d]: with err [%s]", id, i, err)
					if i == q.NumRetries-1 {
						return nil, errors.Errorf("failed to get audit tokens, tx [%s] is still pending", id.TxId)
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

func (q *QueryEngine) ListHistoryIssuedTokens() (*token.IssuedTokens, error) {
	return q.qe.ListHistoryIssuedTokens()
}

// PublicParams returns the public parameters stored in the vault
func (q *QueryEngine) PublicParams() ([]byte, error) {
	return q.qe.PublicParams()
}

// GetTokens returns the tokens stored in the vault matching the given ids
func (q *QueryEngine) GetTokens(inputs ...*token.ID) ([]*token.Token, error) {
	tokens, err := q.qe.GetTokens(inputs...)
	return tokens, err
}

// GetStatus returns the status of the passed transaction
func (q *QueryEngine) GetStatus(txID string) (TxStatus, string, error) {
	return q.qe.GetStatus(txID)
}

type CertificationStorage struct {
	c driver.CertificationStorage
}

func (c *CertificationStorage) Exists(id *token.ID) bool {
	return c.c.Exists(id)
}

func (c *CertificationStorage) Store(certifications map[*token.ID][]byte) error {
	return c.c.Store(certifications)
}

// Vault models a token vault
type Vault struct {
	v      driver.Vault
	logger logging.Logger
}

// NewQueryEngine returns a new query engine
func (v *Vault) NewQueryEngine() *QueryEngine {
	return NewQueryEngine(v.logger, v.v.QueryEngine(), 3, 3*time.Second)
}

func (v *Vault) CertificationStorage() *CertificationStorage {
	return &CertificationStorage{v.v.CertificationStorage()}
}

// UnspentTokensIterator models an iterator over all unspent tokens stored in the vault
type UnspentTokensIterator struct {
	driver.UnspentTokensIterator
}

// Sum  computes the sum of the quantities of the tokens in the iterator.
// Sum closes the iterator at the end of the execution.
func (u *UnspentTokensIterator) Sum(precision uint64) (token.Quantity, error) {
	defer u.Close()
	sum := token.NewZeroQuantity(precision)
	for {
		tok, err := u.Next()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			break
		}

		q, err := token.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		sum = sum.Add(q)
	}

	return sum, nil
}
