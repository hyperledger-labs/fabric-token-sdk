/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	errors2 "errors"

	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	views1 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	if err := p.SDK.Install(); err != nil {
		return err
	}
	if err := p.SDK.Container().Invoke(func(in struct {
		dig.In
		Registry driver.Registry // replace this with an external interface
	}) error {
		return errors2.Join(
			in.Registry.RegisterFactory("registerAuditor", &views.RegisterAuditorViewFactory{}),
			in.Registry.RegisterFactory("historyAuditing", &views.ListAuditedTransactionsViewFactory{}),
			in.Registry.RegisterFactory("holding", &views.CurrentHoldingViewFactory{}),
			in.Registry.RegisterFactory("spending", &views.CurrentSpendingViewFactory{}),
			in.Registry.RegisterFactory("balance", &views.BalanceViewFactory{}),
			in.Registry.RegisterFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}),
			in.Registry.RegisterFactory("SetTransactionAuditStatus", &views.SetTransactionAuditStatusViewFactory{}),
			in.Registry.RegisterFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}),
			in.Registry.RegisterFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}),
			in.Registry.RegisterFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}),
			in.Registry.RegisterFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}),
			in.Registry.RegisterFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{}),
			in.Registry.RegisterFactory("RevokeUser", &views.RevokeUserViewFactory{}),
			in.Registry.RegisterFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}),
			in.Registry.RegisterFactory("TxFinality", &views1.TxFinalityViewFactory{}),
			in.Registry.RegisterFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}),
		)
	}); err != nil {
		return errors.WithMessage(err, "failed to install auditor's views")
	}
	return nil
}
