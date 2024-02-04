/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Vault struct {
	tmsID   token2.TMSID
	tokenDB *tokendb.DB
	ttxdb   *ttxdb.DB

	queryEngine          *QueryEngine
	certificationStorage certification.Storage
}

func NewVault(tmsID token2.TMSID, cs certification.Storage, ttxdb *ttxdb.DB, tokenDB *tokendb.DB, backend driver2.Vault) (*Vault, error) {
	return &Vault{
		tmsID:   tmsID,
		tokenDB: tokenDB,
		ttxdb:   ttxdb,
		queryEngine: &QueryEngine{
			ns:      tmsID.Namespace,
			backend: backend,
			tokenDB: tokenDB,
		},
		certificationStorage: cs,
	}, nil
}

func (v *Vault) QueryEngine() vault.QueryEngine {
	return v.queryEngine
}

func (v *Vault) CertificationStorage() certification.Storage {
	return v.certificationStorage
}

func (v *Vault) DeleteTokens(ids ...*token.ID) error {
	return v.tokenDB.DeleteTokens(ids...)
}

type QueryEngine struct {
	ns      string
	backend driver2.Vault
	tokenDB *tokendb.DB
}

func (q *QueryEngine) IsPending(id *token.ID) (bool, error) {
	vc, err := q.backend.TransactionStatus(id.TxId)
	if err != nil {
		return false, err
	}
	return vc == driver2.Busy, nil
}

func (q *QueryEngine) IsMine(id *token.ID) (bool, error) {
	return q.tokenDB.IsMine(id.TxId, id.Index)
}

func (q *QueryEngine) UnspentTokensIterator() (driver.UnspentTokensIterator, error) {
	return q.tokenDB.UnspentTokensIterator()
}

func (q *QueryEngine) UnspentTokensIteratorBy(id, typ string) (driver.UnspentTokensIterator, error) {
	return q.tokenDB.UnspentTokensIteratorBy(id, typ)
}

func (q *QueryEngine) ListUnspentTokens() (*token.UnspentTokens, error) {
	return q.tokenDB.ListUnspentTokens()
}

func (q *QueryEngine) ListAuditTokens(ids ...*token.ID) ([]*token.Token, error) {
	return q.tokenDB.ListAuditTokens(ids...)
}

func (q *QueryEngine) ListHistoryIssuedTokens() (*token.IssuedTokens, error) {
	return q.tokenDB.ListHistoryIssuedTokens()
}

func (q *QueryEngine) PublicParams() ([]byte, error) {
	return q.tokenDB.GetPublicParams()
}

func (q *QueryEngine) GetTokenInfos(ids []*token.ID, callback driver.QueryCallbackFunc) error {
	return q.tokenDB.GetTokenInfos(ids, callback)
}

func (q *QueryEngine) GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error {
	return q.tokenDB.GetTokenOutputs(ids, callback)
}

func (q *QueryEngine) GetTokenInfoAndOutputs(ids []*token.ID, callback driver.QueryCallback2Func) error {
	return q.tokenDB.GetTokenInfoAndOutputs(ids, callback)
}

func (q *QueryEngine) GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error) {
	return q.tokenDB.GetTokens(inputs...)
}

func (q *QueryEngine) WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error) {
	return q.tokenDB.WhoDeletedTokens(inputs...)
}
