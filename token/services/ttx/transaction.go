/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Transaction struct {
	*endorser.Transaction
	*Namespace
}

func Wrap(context view.Context, tx *endorser.Transaction, opts ...TxOption) (*Transaction, error) {
	namespace, err := NewNamespace(tx, opts...)
	if err != nil {
		return nil, err
	}
	context.OnError(namespace.Release)
	tx.AppendVerifierProvider(&verifierProvider{SignatureService: namespace.TokenService().SigService()})

	return &Transaction{
		Transaction: tx,
		Namespace:   namespace,
	}, nil
}

func NewTransaction(context view.Context, opts ...TxOption) (*Transaction, error) {
	_, tx, err := endorser.NewTransaction(context)
	if err != nil {
		return nil, err
	}

	namespace, err := NewNamespace(tx, opts...)
	if err != nil {
		return nil, err
	}
	namespace.SetProposal()
	context.OnError(namespace.Release)

	return &Transaction{
		Transaction: tx,
		Namespace:   namespace,
	}, nil
}

func NewTransactionFromBytes(context view.Context, bytes []byte) (*Transaction, error) {
	txBuilder := endorser.NewBuilder(context)
	tx, err := txBuilder.NewTransactionFromBytes(bytes)
	if err != nil {
		return nil, err
	}

	namespace, err := NewNamespace(tx)
	if err != nil {
		return nil, err
	}
	context.OnError(namespace.Release)

	return &Transaction{
		Transaction: tx,
		Namespace:   namespace,
	}, nil
}

func ReceiveTransaction(context view.Context) (*Transaction, error) {
	return NewTransactionFromBytes(context, session.ReadFirstMessageOrPanic(context))
}

func NewCollectApprovesView(tx *Transaction, parties ...view.Identity) view.View {
	return endorser.NewCollectApprovesView(tx.tx, parties...)
}

func NewOrderingView(tx *Transaction) view.View {
	return endorser.NewOrderingView(tx.tx)
}

func NewFinalityView(tx *Transaction) view.View {
	return endorser.NewFinalityView(tx.tx)
}

func NewApproveView(tx *Transaction, ids ...view.Identity) view.View {
	return endorser.NewEndorseView(tx.tx, ids...)
}

func NewAcceptView(tx *Transaction) view.View {
	return endorser.NewEndorseView(tx.tx, tx.Receivers()...)
}
