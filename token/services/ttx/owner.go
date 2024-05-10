/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type TxOwner struct {
	tms                     *token.ManagementService
	owner                   *DB
	transactionInfoProvider *TransactionInfoProvider
}

// NewOwner returns a new owner service.
func NewOwner(sp token.ServiceProvider, tms *token.ManagementService) *TxOwner {
	backend := New(sp, tms)
	return &TxOwner{
		tms:                     tms,
		owner:                   backend,
		transactionInfoProvider: newTransactionInfoProvider(tms, backend),
	}
}

// Append adds a new transaction to the token transaction database.
func (a *TxOwner) Append(tx *Transaction) error {
	return a.owner.Append(tx)
}

// Transactions returns an iterators of transaction records filtered by the given params.
func (a *TxOwner) Transactions(params QueryTransactionsParams) (driver.TransactionIterator, error) {
	return a.owner.ttxDB.Transactions(params)
}

// TransactionInfo returns the transaction info for the given transaction ID.
func (a *TxOwner) TransactionInfo(txID string) (*TransactionInfo, error) {
	return a.transactionInfoProvider.TransactionInfo(txID)
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *TxOwner) SetStatus(txID string, status TxStatus, message string) error {
	return a.owner.SetStatus(txID, status, message)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *TxOwner) GetStatus(txID string) (TxStatus, string, error) {
	return a.owner.GetStatus(txID)
}

func (a *TxOwner) GetTokenRequest(txID string) ([]byte, error) {
	return a.owner.GetTokenRequest(txID)
}

func (a *TxOwner) appendTransactionEndorseAck(tx *Transaction, id view.Identity, sigma []byte) error {
	return a.owner.AppendTransactionEndorseAck(tx.ID(), id, sigma)
}
