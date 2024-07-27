/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	TokenRequestToSign driver.ValidationAttributeID = "trs"
)

type Vault struct {
	tmsID   token2.TMSID
	tokenDB *tokendb.DB

	queryEngine          *QueryEngine
	certificationStorage vault.CertificationStorage
}

func NewVault(tmsID token2.TMSID, auditdb *auditdb.DB, ttxdb *ttxdb.DB, tokenDB *tokendb.DB) (*Vault, error) {
	return &Vault{
		tmsID:   tmsID,
		tokenDB: tokenDB,
		queryEngine: &QueryEngine{
			DB:      tokenDB,
			auditDB: auditdb,
			ttxdb:   ttxdb,
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
	auditDB *auditdb.DB
	ttxdb   *ttxdb.DB
}

func (q *QueryEngine) IsPending(id *token.ID) (bool, error) {
	vd, _, err := q.GetStatus(id.TxId)
	if err != nil {
		return false, err
	}
	return vd == ttxdb.Pending, nil
}

func (q *QueryEngine) GetStatus(txID string) (vault.TxStatus, string, error) {
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
