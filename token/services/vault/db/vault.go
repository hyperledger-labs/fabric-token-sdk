/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"runtime/debug"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
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
			backend: backend,
			DB:      tokenDB,
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
	return v.tokenDB.DeleteTokens(string(debug.Stack()), ids...)
}

type QueryEngine struct {
	*tokendb.DB
	backend driver2.Vault
}

func (q *QueryEngine) IsPending(id *token.ID) (bool, error) {
	vc, _, err := q.backend.TransactionStatus(id.TxId)
	if err != nil {
		return false, err
	}
	return vc == driver2.Busy, nil
}

func (q *QueryEngine) IsMine(id *token.ID) (bool, error) {
	if id == nil {
		return false, nil
	}
	return q.DB.IsMine(id.TxId, id.Index)
}
