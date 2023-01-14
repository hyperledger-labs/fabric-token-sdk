/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package owner

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.owner")

type QueryTransactionsParams = ttxdb.QueryTransactionsParams

// TxStatus is the status of a transaction
type TxStatus = ttxdb.TxStatus

const (
	Unknown = ttxdb.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = ttxdb.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = ttxdb.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = ttxdb.Deleted
)

// Transaction models a token transaction
type Transaction interface {
	ID() string
	Network() string
	Channel() string
	Request() *token.Request
}

// QueryExecutor defines the interface for the query executor
type QueryExecutor struct {
	*ttxdb.QueryExecutor
}

// Owner is the interface for the owner service
type Owner struct {
	sp view.ServiceProvider
	db *ttxdb.DB
}

// NewQueryExecutor returns a new query executor
func (a *Owner) NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{QueryExecutor: a.db.NewQueryExecutor()}
}

// Append adds the passed transaction to the database
func (a *Owner) Append(tx Transaction) error {
	// append request to the db
	if err := a.db.AppendTransactionRecord(tx.Request()); err != nil {
		return errors.WithMessagef(err, "failed appending request %s", tx.ID())
	}

	// listen to events
	net := network.GetInstance(a.sp, tx.Network(), tx.Channel())
	if net == nil {
		return errors.Errorf("failed getting network instance for [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("register tx status listener for tx %s at network", tx.ID(), tx.Network())
	if err := net.SubscribeTxStatusChanges(tx.ID(), &TxStatusChangesListener{net, a.db}); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("append done for request %s", tx.ID())
	return nil
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *Owner) SetStatus(txID string, status TxStatus) error {
	return a.db.SetStatus(txID, status)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *Owner) GetStatus(txID string) (TxStatus, error) {
	return a.db.GetStatus(txID)
}

type TxStatusChangesListener struct {
	net *network.Network
	db  *ttxdb.DB
}

func (t *TxStatusChangesListener) OnStatusChange(txID string, status int) error {
	logger.Debugf("tx status changed for tx %s: %s", txID, status)
	var txStatus ttxdb.TxStatus
	switch network.ValidationCode(status) {
	case network.Valid:
		txStatus = ttxdb.Confirmed
	case network.Invalid:
		txStatus = ttxdb.Deleted
	}
	if err := t.db.SetStatus(txID, txStatus); err != nil {
		return errors.WithMessagef(err, "failed setting status for request %s", txID)
	}
	logger.Debugf("tx status changed for tx %s: %s done", txID, status)
	go func() {
		logger.Debugf("unsubscribe for tx %s...", txID)
		if err := t.net.UnsubscribeTxStatusChanges(txID, t); err != nil {
			logger.Errorf("failed to unsubscribe auditor tx listener for tx-id [%s]: [%s]", txID, err)
		}
		logger.Debugf("unsubscribe for tx %s...done", txID)
	}()
	return nil
}
