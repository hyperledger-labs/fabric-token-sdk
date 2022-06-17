/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
)

// ListIssuedTokens contains the input to query the list of issued tokens
type ListIssuedTokens struct {
	// Wallet whose identities own the token
	Wallet string
	// TokenType is the token type to select
	TokenType string
}

type ListIssuedTokensView struct {
	*ListIssuedTokens
}

func (p *ListIssuedTokensView) Call(context view.Context) (interface{}, error) {
	// Tokens issued by identities in this wallet will be listed
	wallet := ttx.GetIssuerWallet(context, p.Wallet)
	assert.NotNil(wallet, "wallet [%s] not found", p.Wallet)

	// Return the list of issued tokens by type
	return wallet.ListIssuedTokens(ttx.WithType(p.TokenType))
}

type ListIssuedTokensViewFactory struct{}

func (i *ListIssuedTokensViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListIssuedTokensView{ListIssuedTokens: &ListIssuedTokens{}}
	err := json.Unmarshal(in, f.ListIssuedTokens)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type ListAuditedTransactions struct {
	From *time.Time
	To   *time.Time
}

type ListAuditedTransactionsView struct {
	*ListAuditedTransactions
}

func (p *ListAuditedTransactionsView) Call(context view.Context) (interface{}, error) {
	// Tokens issued by identities in this wallet will be listed
	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	// Get query executor
	auditor := ttx.NewAuditor(context, w)
	aqe := auditor.NewQueryExecutor()
	defer aqe.Done()
	it, err := aqe.Transactions(ttxdb.QueryTransactionsParams{From: p.From, To: p.To})
	assert.NoError(err, "failed querying transactions")
	defer it.Close()

	// Return the list of audited transactions
	var txs []*ttxdb.TransactionRecord
	for {
		tx, err := it.Next()
		assert.NoError(err, "failed iterating over transactions")
		if tx == nil {
			break
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

type ListAuditedTransactionsViewFactory struct{}

func (i *ListAuditedTransactionsViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListAuditedTransactionsView{ListAuditedTransactions: &ListAuditedTransactions{}}
	err := json.Unmarshal(in, f.ListAuditedTransactions)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

// ListAcceptedTransactions contains the input to query the list of issued tokens
type ListAcceptedTransactions struct {
	SenderWallet    string
	RecipientWallet string
	From            *time.Time
	To              *time.Time
}

type ListAcceptedTransactionsView struct {
	*ListAcceptedTransactions
}

func (p *ListAcceptedTransactionsView) Call(context view.Context) (interface{}, error) {
	// Get query executor
	owner := ttx.NewOwner(context, token.GetManagementService(context))
	aqe := owner.NewQueryExecutor()
	defer aqe.Done()
	it, err := aqe.Transactions(ttxdb.QueryTransactionsParams{
		SenderWallet:    p.SenderWallet,
		RecipientWallet: p.RecipientWallet,
		From:            p.From,
		To:              p.To,
	})
	assert.NoError(err, "failed querying transactions")
	defer it.Close()

	// Return the list of accepted transactions
	var txs []*ttxdb.TransactionRecord
	for {
		tx, err := it.Next()
		assert.NoError(err, "failed iterating over transactions")
		if tx == nil {
			break
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

type ListAcceptedTransactionsViewFactory struct{}

func (i *ListAcceptedTransactionsViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListAcceptedTransactionsView{ListAcceptedTransactions: &ListAcceptedTransactions{}}
	err := json.Unmarshal(in, f.ListAcceptedTransactions)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
