/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"context"
	"runtime/debug"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
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

func (v *Vault) DeleteTokens(ctx context.Context, ids ...*token.ID) error {
	return v.tokenDB.DeleteTokens(ctx, string(debug.Stack()), ids...)
}

type QueryEngine struct {
	*tokendb.StoreService
	auditDB *auditdb.StoreService
	ttxdb   *ttxdb.StoreService
}

func (q *QueryEngine) IsPending(ctx context.Context, id *token.ID) (bool, error) {
	vd, _, err := q.GetStatus(ctx, id.TxId)
	if err != nil {
		return false, err
	}

	return vd == ttxdb.Pending, nil
}

func (q *QueryEngine) GetStatus(ctx context.Context, txID string) (driver.TxStatus, string, error) {
	vd, msg, err := q.ttxdb.GetStatus(ctx, txID)
	if err != nil || vd == ttxdb.Unknown {
		vd, msg, err = q.auditDB.GetStatus(ctx, txID)
	}

	return vd, msg, err
}

func (q *QueryEngine) IsMine(ctx context.Context, id *token.ID) (bool, error) {
	if id == nil {
		return false, nil
	}

	return q.StoreService.IsMine(ctx, id.TxId, id.Index)
}

type CertificationStorage struct {
	*tokendb.StoreService
}

func (t *CertificationStorage) Exists(ctx context.Context, id *token.ID) bool {
	return t.ExistsCertification(ctx, id)
}

func (t *CertificationStorage) Store(ctx context.Context, certifications map[*token.ID][]byte) error {
	return t.StoreCertifications(ctx, certifications)
}
