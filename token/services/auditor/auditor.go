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

// Payments returns a filter for payments
func (a *QueryExecutor) Payments() *ttxdb.PaymentsFilter {
	return a.QueryExecutor.NewPaymentsFilter()
}

// Holdings returns a filter for holdings
func (a *QueryExecutor) Holdings() *ttxdb.HoldingsFilter {
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

// New returns a new Auditor instance for the passed auditor wallet
func New(sp view2.ServiceProvider, w *token.AuditorWallet) *Auditor {
	return &Auditor{sp: sp, db: ttxdb.Get(sp, w)}
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

type TxStatusChangesListener struct {
	net *network.Network
	db  *ttxdb.DB
}

func (t *TxStatusChangesListener) OnStatusChange(txID string, status int) error {
	logger.Debugf("tx status changed for tx %s: %s", txID, status)
	var auditDBTxStatus ttxdb.TxStatus
	switch network.ValidationCode(status) {
	case network.Valid:
		auditDBTxStatus = ttxdb.Confirmed
	case network.Invalid:
		auditDBTxStatus = ttxdb.Deleted
	}
	if err := t.db.SetStatus(txID, auditDBTxStatus); err != nil {
		return errors.WithMessagef(err, "failed setting status for request %s", txID)
	}
	logger.Debugf("tx status changed for tx %s: %s done", txID, status)
	return nil
}
