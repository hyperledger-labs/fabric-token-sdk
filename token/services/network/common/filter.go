/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

type TransactionFilterProvider[F driver2.TransactionFilter] interface {
	New(tmsID token3.TMSID) (F, error)
}

// AcceptTxInDBFilterProvider provides instances of AcceptTxInDBsFilter based on the transaction db and audit db
// for a given TMS
type AcceptTxInDBFilterProvider struct {
	ttxDBProvider   *ttxdb.Manager
	auditDBProvider *auditdb.Manager
}

func NewAcceptTxInDBFilterProvider(ttxDBProvider *ttxdb.Manager, auditDBProvider *auditdb.Manager) *AcceptTxInDBFilterProvider {
	return &AcceptTxInDBFilterProvider{ttxDBProvider: ttxDBProvider, auditDBProvider: auditDBProvider}
}

func (p *AcceptTxInDBFilterProvider) New(tmsID token3.TMSID) (*AcceptTxInDBsFilter, error) {
	ttxDB, err := p.ttxDBProvider.ServiceByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get transaction db for [%s]", tmsID)
	}
	auditDB, err := p.auditDBProvider.ServiceByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get audit db for [%s]", tmsID)
	}
	return &AcceptTxInDBsFilter{
		ttxDB:   ttxDB,
		auditDB: auditDB,
	}, nil
}

// AcceptTxInDBsFilter uses the transaction db and the audit db to decide if a given transaction needs
// to be further processed by the token-sdk upon a network event about its finality
type AcceptTxInDBsFilter struct {
	ttxDB   *ttxdb.StoreService
	auditDB *auditdb.StoreService
}

func (t *AcceptTxInDBsFilter) Accept(txID string, env []byte) (bool, error) {
	status, _, err := t.ttxDB.GetStatus(txID)
	if err != nil {
		return false, err
	}
	if status != ttxdb.Unknown {
		return true, nil
	}
	status, _, err = t.auditDB.GetStatus(txID)
	if err != nil {
		return false, err
	}
	return status != auditdb.Unknown, nil
}
