/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryEngine struct {
	qe driver.QueryEngine
}

func (q *QueryEngine) IsMine(id *token2.ID) (bool, error) {
	return q.qe.IsMine(id)
}

func (q *QueryEngine) UnspentTokensIterator() (*UnspentTokensIterator, error) {
	it, err := q.qe.UnspentTokensIterator()
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{UnspentTokensIterator: it}, nil
}

func (q *QueryEngine) UnspentTokensIteratorBy(id, typ string) (*UnspentTokensIterator, error) {
	it, err := q.qe.UnspentTokensIteratorBy(id, typ)
	if err != nil {
		return nil, err
	}
	return &UnspentTokensIterator{UnspentTokensIterator: it}, nil
}

func (q *QueryEngine) ListUnspentTokens() (*token2.UnspentTokens, error) {
	return q.qe.ListUnspentTokens()
}

func (q *QueryEngine) ListAuditTokens(ids ...*token2.ID) ([]*token2.Token, error) {
	return q.qe.ListAuditTokens(ids...)
}

func (q *QueryEngine) ListHistoryIssuedTokens() (*token2.IssuedTokens, error) {
	return q.qe.ListHistoryIssuedTokens()
}

func (q *QueryEngine) PublicParams() ([]byte, error) {
	return q.qe.PublicParams()
}

func (q *QueryEngine) GetTokens(inputs ...*token2.ID) ([]*token2.Token, error) {
	_, tokens, err := q.qe.GetTokens(inputs...)
	return tokens, err
}

type Vault struct {
	v driver.Vault
}

func (v *Vault) NewQueryEngine() *QueryEngine {
	return &QueryEngine{
		qe: v.v.QueryEngine(),
	}
}

type UnspentTokensIterator struct {
	driver.UnspentTokensIterator
}
