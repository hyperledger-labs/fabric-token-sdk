/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/driver"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	"github.com/pkg/errors"
)

type StatusFetcher struct {
	dbManager *DBManager
}

func NewStatusFetcher(dbManager *DBManager) *StatusFetcher {
	return &StatusFetcher{dbManager: dbManager}
}

func (r *StatusFetcher) FetchCode(network string, txID driver.TxID) (*orion.Session, driver2.ValidationCode, error) {
	sm, err := r.dbManager.GetSessionManager(network)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to get session manager for network [%s]", network)
	}
	oSession, err := sm.GetSession()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to create session to orion network [%s]", network)
	}
	ledger, err := oSession.Ledger()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to get ledger for orion network [%s]", network)
	}
	tx, err := ledger.GetTransactionByID(txID)
	if err != nil {
		if errors2.HasType(err, &bcdb.ErrorNotFound{}) {
			return oSession, driver2.Unknown, nil
		}
		return nil, 0, errors.Wrapf(err, "failed to get transaction [%s] for orion network [%s]", txID, network)
	}
	if tx.ValidationCode() == orion.VALID {
		return oSession, driver2.Valid, nil
	}
	return oSession, driver2.Invalid, nil
}
