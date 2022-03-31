/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// QueryEngine models a token query engine
type QueryEngine struct {
	qe driver.QueryEngine
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
	return q.qe.ListAuditTokens(ids...)
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
	return &QueryEngine{
		qe: v.v.QueryEngine(),
	}
}

// UnspentTokensIterator models an iterator over all unspent tokens stored in the vault
type UnspentTokensIterator struct {
	driver.UnspentTokensIterator
}
