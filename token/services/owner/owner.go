/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package owner

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.owner")

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

// NewPaymentsFilter returns a new filter for payments
func (a *QueryExecutor) NewPaymentsFilter() *ttxdb.PaymentsFilter {
	return a.QueryExecutor.NewPaymentsFilter()
}

// NewHoldingsFilter returns a new filter for holdings
func (a *QueryExecutor) NewHoldingsFilter() *ttxdb.HoldingsFilter {
	return a.QueryExecutor.NewHoldingsFilter()
}

// Done closes the query executor. It must be called when the query executor is no longer needed.
func (a *QueryExecutor) Done() {
	a.QueryExecutor.Done()
}

// Owner is the interface for the owner service
type Owner struct {
	sp view2.ServiceProvider
	db *ttxdb.DB
}

// New returns a new Owner instance for the passed wallet
func New(sp view2.ServiceProvider, tms *token.ManagementService) *Owner {
	return &Owner{sp: sp, db: ttxdb.Get(sp, &tmsWallet{tms: tms})}
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
	return nil
}

type tmsWallet struct {
	tms *token.ManagementService
}

func (t *tmsWallet) ID() string {
	return ""
}

func (t *tmsWallet) TMS() *token.ManagementService {
	return t.tms
}
