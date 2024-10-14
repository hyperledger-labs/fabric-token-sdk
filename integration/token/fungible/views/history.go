/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

// ListIssuedTokens contains the input to query the list of issued tokens
type ListIssuedTokens struct {
	// Wallet whose identities own the token
	Wallet string
	// TokenType is the token type to select
	TokenType string
	// The TMS to pick in case of multiple TMSIDs
	TMSID *token.TMSID
}

type ListIssuedTokensView struct {
	*ListIssuedTokens
}

func (p *ListIssuedTokensView) Call(context view.Context) (interface{}, error) {
	// Tokens issued by identities in this wallet will be listed
	wallet := ttx.GetIssuerWallet(context, p.Wallet, ServiceOpts(p.TMSID)...)
	if wallet == nil {
		return nil, errors.Errorf("wallet [%s] not found", p.Wallet)
	}

	// Return the list of issued tokens by type
	return wallet.ListIssuedTokens(ttx.WithType(p.TokenType))
}

type ListIssuedTokensViewFactory struct{}

func (i *ListIssuedTokensViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListIssuedTokensView{ListIssuedTokens: &ListIssuedTokens{}}
	if err := json.Unmarshal(in, f.ListIssuedTokens); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling input")
	}
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
	if w == nil {
		return nil, errors.New("failed getting default auditor wallet")
	}

	// Get query executor
	auditor, err := ttx.NewAuditor(context, w)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get auditor instance")
	}

	it, err := auditor.Transactions(ttxdb.QueryTransactionsParams{From: p.From, To: p.To})
	if err != nil {
		return nil, errors.Wrapf(err, "failed querying transactions")
	}
	return ToSlice(it)
}

type ListAuditedTransactionsViewFactory struct{}

func (i *ListAuditedTransactionsViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListAuditedTransactionsView{ListAuditedTransactions: &ListAuditedTransactions{}}
	if err := json.Unmarshal(in, f.ListAuditedTransactions); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling input")
	}
	return f, nil
}

// ListAcceptedTransactions contains the input to query the list of accepted tokens
type ListAcceptedTransactions struct {
	SenderWallet    string
	RecipientWallet string
	From            *time.Time
	To              *time.Time
	ActionTypes     []ttxdb.ActionType
	Statuses        []ttxdb.TxStatus
	TMSID           *token.TMSID
}

type ListAcceptedTransactionsView struct {
	*ListAcceptedTransactions
}

func (p *ListAcceptedTransactionsView) Call(context view.Context) (interface{}, error) {
	// Get query executor
	owner := ttx.NewOwner(context, token.GetManagementService(context, ServiceOpts(p.TMSID)...))
	it, err := owner.Transactions(ttxdb.QueryTransactionsParams{
		SenderWallet:    p.SenderWallet,
		RecipientWallet: p.RecipientWallet,
		From:            p.From,
		To:              p.To,
		ActionTypes:     p.ActionTypes,
		Statuses:        p.Statuses,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed querying transactions")
	}

	return ToSlice(it)
}

type ListAcceptedTransactionsViewFactory struct{}

func (l *ListAcceptedTransactionsViewFactory) NewView(in []byte) (view.View, error) {
	v := &ListAcceptedTransactionsView{ListAcceptedTransactions: &ListAcceptedTransactions{}}
	if err := json.Unmarshal(in, v.ListAcceptedTransactions); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling input")
	}
	return v, nil
}

// TransactionInfo contains the input information to search for transaction info
type TransactionInfo struct {
	TransactionID string
	TMSID         *token.TMSID
}

type TransactionInfoView struct {
	*TransactionInfo
}

func (t *TransactionInfoView) Call(context view.Context) (interface{}, error) {
	owner := ttx.NewOwner(context, token.GetManagementService(context, ServiceOpts(t.TMSID)...))
	info, err := owner.TransactionInfo(t.TransactionID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting transaction info")
	}

	return info, nil
}

type TransactionInfoViewFactory struct{}

func (p *TransactionInfoViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransactionInfoView{TransactionInfo: &TransactionInfo{}}
	if err := json.Unmarshal(in, f.TransactionInfo); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling input")
	}
	return f, nil
}

func ToSlice[T any](it collections.Iterator[*T]) ([]*T, error) {
	defer it.Close()
	var items []*T
	for {
		if tx, err := it.Next(); err != nil {
			return nil, errors.Wrapf(err, "failed iterating over transactions")
		} else if tx == nil {
			break
		} else {
			items = append(items, tx)
		}
	}
	return items, nil
}
