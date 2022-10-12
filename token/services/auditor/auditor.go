/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.auditor")

// TxStatus is the status of a transaction
type TxStatus = ttxdb.TxStatus

const (
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = ttxdb.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = ttxdb.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = ttxdb.Deleted
)

// Transaction models a generic token transaction
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

// NewPaymentsFilter returns a filter for payments
func (a *QueryExecutor) NewPaymentsFilter() *ttxdb.PaymentsFilter {
	return a.QueryExecutor.NewPaymentsFilter()
}

// NewHoldingsFilter returns a filter for holdings
func (a *QueryExecutor) NewHoldingsFilter() *ttxdb.HoldingsFilter {
	return a.QueryExecutor.NewHoldingsFilter()
}

// Done closes the query executor. It must be called when the query executor is no longer needed.
func (a *QueryExecutor) Done() {
	a.QueryExecutor.Done()
}

// Auditor is the interface for the auditor service
type Auditor struct {
	sp view2.ServiceProvider
	db *ttxdb.DB
}

// Validate validates the passed token request
func (a *Auditor) Validate(request *token.Request) error {
	return request.AuditCheck()
}

// Audit evaluates the passed token request and returns the list on inputs and outputs in the request
func (a *Auditor) Audit(request *token.Request) (*token.InputStream, *token.OutputStream, error) {
	inputs, err := request.AuditInputs()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting inputs")
	}
	outputs, err := request.AuditOutputs()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting outputs")
	}

	return inputs, outputs, nil
}

// NewQueryExecutor returns a new query executor
func (a *Auditor) NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{QueryExecutor: a.db.NewQueryExecutor()}
}

// Append adds the passed transaction to the auditor database
func (a *Auditor) Append(tx Transaction) error {
	// append request to audit db
	if err := a.db.Append(tx.Request()); err != nil {
		return errors.WithMessagef(err, "failed appending request %s", tx.ID())
	}

	// lister to events
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
func (a *Auditor) SetStatus(txID string, status TxStatus) error {
	return a.db.SetStatus(txID, status)
}

type TxStatusChangesListener struct {
	net *network.Network
	db  *ttxdb.DB
}

func (t *TxStatusChangesListener) OnStatusChange(txID string, status int) error {
	logger.Infof("tx status changed for tx %s: %s", txID, status)
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
