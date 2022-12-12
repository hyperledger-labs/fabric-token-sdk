/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// Transaction holds a ttx transaction
type Transaction struct {
	*ttx.Transaction
}

// NewAnonymousTransaction returns a new anonymous token transaction customized with the passed opts
func NewAnonymousTransaction(sp view.Context, opts ...ttx.TxOption) (*Transaction, error) {
	tx, err := ttx.NewAnonymousTransaction(sp, opts...)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		Transaction: tx,
	}, nil
}

// NewTransactionFromBytes returns a new transaction from the passed bytes
func NewTransactionFromBytes(ctx view.Context, raw []byte) (*Transaction, error) {
	tx, err := ttx.NewTransactionFromBytes(ctx, raw)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		Transaction: tx,
	}, nil
}

// Outputs returns a new OutputStream of the transaction's outputs
func (t *Transaction) Outputs() (*OutputStream, error) {
	outs, err := t.TokenRequest.Outputs()
	if err != nil {
		return nil, err
	}
	return NewOutputStream(outs), nil
}
