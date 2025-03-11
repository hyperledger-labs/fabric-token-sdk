/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issuer

import (
	errors2 "errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	views1 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewSDK(registry)}
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	if err := p.SDK.Install(); err != nil {
		return err
	}

	if err := p.Container().Invoke(func(in struct {
		dig.In
		Registry driver.Registry // replace this with an external interface
	}) error {
		return errors2.Join(
			in.Registry.RegisterFactory("issue", &views.IssueCashViewFactory{}),
			in.Registry.RegisterFactory("transfer", &views.TransferViewFactory{}),
			in.Registry.RegisterFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{}),
			in.Registry.RegisterFactory("redeem", &views.RedeemViewFactory{}),
			in.Registry.RegisterFactory("balance", &views.BalanceViewFactory{}),
			in.Registry.RegisterFactory("historyIssuedToken", &views.ListIssuedTokensViewFactory{}),
			in.Registry.RegisterFactory("issuedTokenQuery", &views.ListIssuedTokensViewFactory{}),
			in.Registry.RegisterFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{}),
			in.Registry.RegisterFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{}),
			in.Registry.RegisterFactory("transactionInfo", &views.TransactionInfoViewFactory{}),
			in.Registry.RegisterFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}),
			in.Registry.RegisterFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}),
			in.Registry.RegisterFactory("RegisterIssuerIdentity", &views.RegisterIssuerIdentityViewFactory{}),
			in.Registry.RegisterFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}),
			in.Registry.RegisterFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}),
			in.Registry.RegisterFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}),
			in.Registry.RegisterFactory("SetKVSEntry", &views.SetKVSEntryViewFactory{}),
			in.Registry.RegisterFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}),
			in.Registry.RegisterFactory("TxFinality", &views1.TxFinalityViewFactory{}),
			in.Registry.RegisterFactory("issue", &views.IssueCashViewFactory{}),
			in.Registry.RegisterResponder(&views.WithdrawalResponderView{}, &views.WithdrawalInitiatorView{}),
			in.Registry.RegisterResponder(&views.TokensUpgradeResponderView{}, &views.TokensUpgradeInitiatorView{}),
		)
	}); err != nil {
		return errors.WithMessage(err, "failed to install issuer's views")
	}
	return nil
}
