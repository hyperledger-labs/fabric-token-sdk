/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"runtime/debug"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Vault struct {
	tmsID   token2.TMSID
	tokenDB *tokendb.StoreService

	queryEngine          *QueryEngine
	certificationStorage driver.CertificationStorage
}

func NewVault(tmsID token2.TMSID, auditdb *auditdb.StoreService, ttxdb *ttxdb.StoreService, tokenDB *tokendb.StoreService) (*Vault, error) {
	return &Vault{
		tmsID:   tmsID,
		tokenDB: tokenDB,
		queryEngine: &QueryEngine{
			StoreService: tokenDB,
			auditDB:      auditdb,
			ttxdb:        ttxdb,
		},
		certificationStorage: &CertificationStorage{StoreService: tokenDB},
	}, nil
}

func (v *Vault) QueryEngine() driver.QueryEngine {
	return v.queryEngine
}

func (v *Vault) CertificationStorage() driver.CertificationStorage {
	return v.certificationStorage
}

func (v *Vault) DeleteTokens(ids ...*token.ID) error {
	return v.tokenDB.DeleteTokens(string(debug.Stack()), ids...)
}

type QueryEngine struct {
	*tokendb.StoreService
	auditDB *auditdb.StoreService
	ttxdb   *ttxdb.StoreService
}

func (q *QueryEngine) IsPending(id *token.ID) (bool, error) {
	vd, _, err := q.GetStatus(id.TxId)
	if err != nil {
		return false, err
	}
	return vd == ttxdb.Pending, nil
}

func (q *QueryEngine) GetStatus(txID string) (driver.TxStatus, string, error) {
	vd, msg, err := q.ttxdb.GetStatus(txID)
	if err != nil || vd == ttxdb.Unknown {
		vd, msg, err = q.auditDB.GetStatus(txID)
	}
	return vd, msg, err
}

func (q *QueryEngine) IsMine(id *token.ID) (bool, error) {
	if id == nil {
		return false, nil
	}
	return q.StoreService.IsMine(id.TxId, id.Index)
}

type CertificationStorage struct {
	*tokendb.StoreService
}

func (t *CertificationStorage) Exists(id *token.ID) bool {
	return t.StoreService.ExistsCertification(id)
}

func (t *CertificationStorage) Store(certifications map[*token.ID][]byte) error {
	return t.StoreService.StoreCertifications(certifications)
}
