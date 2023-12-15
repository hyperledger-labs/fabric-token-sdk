/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
)

type TxOwner struct {
	sp                      view2.ServiceProvider
	tms                     *token.ManagementService
	owner                   *owner.Owner
	transactionInfoProvider *TransactionInfoProvider
}

// NewOwner returns a new owner service.
func NewOwner(sp view2.ServiceProvider, tms *token.ManagementService) *TxOwner {
	backend := owner.New(sp, tms)
	return &TxOwner{
		sp:                      sp,
		tms:                     tms,
		owner:                   backend,
		transactionInfoProvider: newTransactionInfoProvider(sp, tms, backend),
	}
}

// NewQueryExecutor returns a new query executor.
// The query executor is used to execute queries against the token transaction DB.
// The function `Done` on the query executor must be called when it is no longer needed.
func (a *TxOwner) NewQueryExecutor() *owner.QueryExecutor {
	return a.owner.NewQueryExecutor()
}

// Append adds a new transaction to the token transaction database.
func (a *TxOwner) Append(tx *Transaction) error {
	return a.owner.Append(tx)
}

// TransactionInfo returns the transaction info for the given transaction ID.
func (a *TxOwner) TransactionInfo(txID string) (*TransactionInfo, error) {
	return a.transactionInfoProvider.TransactionInfo(txID)
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *TxOwner) SetStatus(txID string, status TxStatus) error {
	return a.owner.SetStatus(txID, status)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *TxOwner) GetStatus(txID string) (TxStatus, error) {
	return a.owner.GetStatus(txID)
}

func (a *TxOwner) GetTokenRequest(txID string) ([]byte, error) {
	return a.owner.GetTokenRequest(txID)
}

func (a *TxOwner) appendTransactionEndorseAck(tx *Transaction, id view.Identity, sigma []byte) error {
	return a.owner.AppendTransactionEndorseAck(tx.ID(), id, sigma)
}
