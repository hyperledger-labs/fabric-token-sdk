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

func (q *QueryEngine) IsMine(id *token2.Id) (bool, error) {
	return q.qe.IsMine(id)
}

func (q *QueryEngine) ListUnspentTokens() (*token2.UnspentTokens, error) {
	return q.qe.ListUnspentTokens()
}

func (q *QueryEngine) ListAuditTokens(ids ...*token2.Id) ([]*token2.Token, error) {
	return q.qe.ListAuditTokens(ids...)
}

func (q *QueryEngine) ListHistoryIssuedTokens() (*token2.IssuedTokens, error) {
	return q.qe.ListHistoryIssuedTokens()
}

func (q *QueryEngine) PublicParams() ([]byte, error) {
	return q.qe.PublicParams()
}

func (q *QueryEngine) GetTokens(inputs ...*token2.Id) ([]*token2.Token, error) {
	return q.qe.GetTokens(inputs...)
}

type Vault struct {
	v driver.Vault
}

func (v *Vault) NewQueryEngine() *QueryEngine {
	return &QueryEngine{
		qe: v.v.QueryEngine(),
	}
}
