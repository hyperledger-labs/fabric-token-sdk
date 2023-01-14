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

// Audit extracts the list of inputs and outputs from the passed transaction.
// In addition, the Audit locks the enrollment named ids.
// Release must be invoked in case
func (a *Auditor) Audit(tx Transaction) (*token.InputStream, *token.OutputStream, error) {
	request := tx.Request()
	record, err := request.AuditRecord()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting transaction audit record")
	}

	var eids []string
	eids = append(eids, record.Inputs.EnrollmentIDs()...)
	eids = append(eids, record.Outputs.EnrollmentIDs()...)
	if err := a.db.AcquireLocks(request.Anchor, eids...); err != nil {
		return nil, nil, err
	}

	return record.Inputs, record.Outputs, nil
}

// Append adds the passed transaction to the auditor database.
// It also releases the locks acquired by Audit.
func (a *Auditor) Append(tx Transaction) error {
	defer a.Release(tx)

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

// Release releases the lock acquired of the passed transaction.
func (a *Auditor) Release(tx Transaction) {
	a.db.ReleaseLocks(tx.Request().Anchor)
}

// NewQueryExecutor returns a new query executor
func (a *Auditor) NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{QueryExecutor: a.db.NewQueryExecutor()}
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *Auditor) SetStatus(txID string, status TxStatus) error {
	return a.db.SetStatus(txID, status)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *Auditor) GetStatus(txID string) (TxStatus, error) {
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
