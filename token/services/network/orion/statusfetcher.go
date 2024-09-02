/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"encoding/base64"
	"time"

	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var runner = db.NewRetryRunner(3, 2*time.Second, true)

type StatusFetcher struct {
	dbManager *DBManager
}

func NewStatusFetcher(dbManager *DBManager) *StatusFetcher {
	return &StatusFetcher{dbManager: dbManager}
}

func (r *StatusFetcher) FetchStatus(network, namespace string, txID driver.TxID) (*TxStatusResponse, error) {
	oSession, code, err := r.fetchCode(network, txID)
	if err != nil {
		return nil, err
	}

	// fetch token request reference
	key, err := keys.CreateTokenRequestKey(txID)
	if err != nil {
		return nil, errors.Errorf("can't create for token request '%s'", txID)
	}

	var trRef []byte
	err = runner.Run(func() error {
		tx, err := oSession.DataTx(txID)
		if err != nil {
			return errors.Wrapf(err, "data tx does not exist for [%s]", txID)
		}
		reqKey := orionKey(key)
		h, err := tx.Get(namespace, reqKey)
		if err != nil {
			return errors.Wrapf(err, "data doesn't exist for [%s]", txID)
		}
		if code == driver2.Valid && len(h) == 0 {
			return errors.Errorf("hash not found for valid transaction [%s:%s,%s,%s]", reqKey, network, namespace, txID)
		}
		trRef = h
		return nil
	})
	if err != nil {
		return nil, err
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("retrieved token request hash for [%s][%s]:[%s]", key, txID, base64.StdEncoding.EncodeToString(trRef))
	}
	return &TxStatusResponse{
		TokenRequestReference: trRef,
		Status:                code,
	}, nil
}

func (r *StatusFetcher) FetchCode(network string, txID driver.TxID) (driver2.ValidationCode, error) {
	_, code, err := r.fetchCode(network, txID)
	return code, err
}

func (r *StatusFetcher) fetchCode(network string, txID driver.TxID) (*orion.Session, driver2.ValidationCode, error) {
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
