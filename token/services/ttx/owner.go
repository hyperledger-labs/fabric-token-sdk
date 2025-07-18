/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
)

type TxOwner struct {
	tms                     *token.ManagementService
	owner                   *Service
	transactionInfoProvider *TransactionInfoProvider
}

// NewOwner returns a new owner service.
func NewOwner(sp token.ServiceProvider, tms *token.ManagementService) *TxOwner {
	backend := New(sp, tms)
	return NewTxOwner(tms, backend)
}

func NewTxOwner(tms *token.ManagementService, backend *Service) *TxOwner {
	return &TxOwner{
		tms:                     tms,
		owner:                   backend,
		transactionInfoProvider: newTransactionInfoProvider(tms, backend),
	}
}

// Append adds a new transaction to the token transaction database.
func (a *TxOwner) Append(ctx context.Context, tx *Transaction) error {
	return a.owner.Append(ctx, tx)
}

// Transactions returns an iterators of transaction records filtered by the given params.
func (a *TxOwner) Transactions(ctx context.Context, params QueryTransactionsParams, pagination driver2.Pagination) (*driver2.PageIterator[*driver.TransactionRecord], error) {
	return a.owner.ttxStoreService.Transactions(ctx, params, pagination)
}

// TransactionInfo returns the transaction info for the given transaction ID.
func (a *TxOwner) TransactionInfo(ctx context.Context, txID string) (*TransactionInfo, error) {
	return a.transactionInfoProvider.TransactionInfo(ctx, txID)
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *TxOwner) SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) error {
	return a.owner.SetStatus(ctx, txID, status, message)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *TxOwner) GetStatus(ctx context.Context, txID string) (TxStatus, string, error) {
	return a.owner.GetStatus(ctx, txID)
}

func (a *TxOwner) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	return a.owner.GetTokenRequest(ctx, txID)
}

func (a *TxOwner) Check(ctx context.Context) ([]string, error) {
	return a.owner.Check(ctx)
}

func (a *TxOwner) appendTransactionEndorseAck(ctx context.Context, tx *Transaction, id view.Identity, sigma []byte) error {
	return a.owner.AppendTransactionEndorseAck(ctx, tx.ID(), id, sigma)
}
