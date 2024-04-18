/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

type QueryTransactionsParams = ttxdb.QueryTransactionsParams

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

// DB is the interface for the owner service
type DB struct {
	networkProvider NetworkProvider
	db              *ttxdb.DB
}

// Append adds the passed transaction to the database
func (a *DB) Append(tx *Transaction) error {
	// append request to the db
	if err := a.db.AppendTransactionRecord(tx.Request()); err != nil {
		return errors.WithMessagef(err, "failed appending request %s", tx.ID())
	}

	// listen to events
	net, err := a.networkProvider.GetNetwork(tx.Network(), tx.Channel())
	if err != nil {
		return errors.WithMessagef(err, "failed getting network instance for [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("register tx status listener for tx [%s:%s] at network", tx.ID(), tx.Network())
	if err := net.AddFinalityListener(tx.ID(), &FinalityListener{net, a.db}); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("append done for request %s", tx.ID())
	return nil
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *DB) SetStatus(txID string, status TxStatus, message string) error {
	return a.db.SetStatus(txID, status, message)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *DB) GetStatus(txID string) (TxStatus, string, error) {
	st, message, err := a.db.GetStatus(txID)
	if err != nil {
		return Unknown, "", err
	}
	return st, message, nil
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (a *DB) GetTokenRequest(txID string) ([]byte, error) {
	return a.db.GetTokenRequest(txID)
}

func (a *DB) AppendTransactionEndorseAck(txID string, id view2.Identity, sigma []byte) error {
	return a.db.AddTransactionEndorsementAck(txID, id, sigma)
}

func (a *DB) GetTransactionEndorsementAcks(id string) (map[string][]byte, error) {
	return a.db.GetTransactionEndorsementAcks(id)
}

type FinalityListener struct {
	net *network.Network
	db  *ttxdb.DB
}

func (t *FinalityListener) OnStatus(txID string, status int, message string) {
	logger.Debugf("tx status changed for tx %s: %s", txID, status)
	var txStatus ttxdb.TxStatus
	switch status {
	case network.Valid:
		txStatus = ttxdb.Confirmed
	case network.Invalid:
		txStatus = ttxdb.Deleted
	}
	if err := t.db.SetStatus(txID, txStatus, message); err != nil {
		logger.Errorf("failed setting status for request [%s]: [%s]", txID, err)
		return
	}
	logger.Debugf("tx status changed for tx %s: %s done", txID, status)
	go func() {
		logger.Debugf("unsubscribe for tx %s...", txID)
		if err := t.net.RemoveFinalityListener(txID, t); err != nil {
			logger.Errorf("failed to unsubscribe auditor tx listener for tx-id [%s]: [%s]", txID, err)
		}
		logger.Debugf("unsubscribe for tx %s...done", txID)
	}()
}
