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
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ValidationCode = int

const (
	_               ValidationCode = iota
	Valid                          // Transaction is valid and committed
	Invalid                        // Transaction is invalid and has been discarded
	Busy                           // Transaction does not yet have a validity state
	Unknown                        // Transaction is unknown
	HasDependencies                // Transaction is unknown but has known dependencies
)

type BackendVault interface {
	TransactionStatus(txID string) (ValidationCode, string, error)
}

type Vault struct {
	tmsID   token2.TMSID
	tokenDB *tokendb.DB
	ttxdb   *ttxdb.DB

	queryEngine          *QueryEngine
	certificationStorage vault.CertificationStorage
}

func NewVault(tmsID token2.TMSID, ttxdb *ttxdb.DB, tokenDB *tokendb.DB, backend BackendVault) (*Vault, error) {
	return &Vault{
		tmsID:   tmsID,
		tokenDB: tokenDB,
		ttxdb:   ttxdb,
		queryEngine: &QueryEngine{
			backend: backend,
			DB:      tokenDB,
		},
		certificationStorage: &CertificationStorage{DB: tokenDB},
	}, nil
}

func (v *Vault) QueryEngine() vault.QueryEngine {
	return v.queryEngine
}

func (v *Vault) CertificationStorage() vault.CertificationStorage {
	return v.certificationStorage
}

func (v *Vault) DeleteTokens(ids ...*token.ID) error {
	return v.tokenDB.DeleteTokens(string(debug.Stack()), ids...)
}

type QueryEngine struct {
	*tokendb.DB
	backend BackendVault
}

func (q *QueryEngine) IsPending(id *token.ID) (bool, error) {
	vc, _, err := q.backend.TransactionStatus(id.TxId)
	if err != nil {
		return false, err
	}
	return vc == Busy, nil
}

func (q *QueryEngine) IsMine(id *token.ID) (bool, error) {
	if id == nil {
		return false, nil
	}
	return q.DB.IsMine(id.TxId, id.Index)
}

type CertificationStorage struct {
	*tokendb.DB
}

func (t *CertificationStorage) Exists(id *token.ID) bool {
	return t.DB.ExistsCertification(id)
}

func (t *CertificationStorage) Store(certifications map[*token.ID][]byte) error {
	return t.DB.StoreCertifications(certifications)
}
