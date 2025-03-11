/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package party

import (
	"context"
	errors2 "errors"
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	views1 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
	registry node.Registry
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{registry: registry}
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	// get dig from registry, this was installed by the FTS's sdk
	var digContainer *dig.Container
	if p.SDK != nil {
		if err := p.SDK.Install(); err != nil {
			return err
		}
		digContainer = p.SDK.Container()
	} else {
		// get it from the registry
		if p.registry == nil {
			return errors.New("registry not set")
		}
		wrapper, err := p.registry.GetService(reflect.TypeOf((*dig.Container)(nil)))
		if err != nil {
			return errors.WithMessage(err, "failed getting dig container service")
		}
		container, ok := wrapper.(*dig.Container)
		if !ok {
			return errors.New("failed getting dig container service, not an instance of *dig.Container")
		}
		digContainer = container
	}

	if err := digContainer.Invoke(func(in struct {
		dig.In
		Registry driver.Registry // replace this with an external interface
	}) error {
		return errors2.Join(
			in.Registry.RegisterFactory("MultiSigLock", &views.MultiSigLockViewFactory{}),
			in.Registry.RegisterFactory("MultiSigSpend", &views.MultiSigSpendViewFactory{}),
			in.Registry.RegisterFactory("CoOwnedBalance", &views.CoOwnedBalanceViewFactory{}),
			in.Registry.RegisterFactory("transfer", &views.TransferViewFactory{}),
			in.Registry.RegisterFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{}),
			in.Registry.RegisterFactory("redeem", &views.RedeemViewFactory{}),
			in.Registry.RegisterFactory("swap", &views.SwapInitiatorViewFactory{}),
			in.Registry.RegisterFactory("history", &views.ListUnspentTokensViewFactory{}),
			in.Registry.RegisterFactory("balance", &views.BalanceViewFactory{}),
			in.Registry.RegisterFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{}),
			in.Registry.RegisterFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{}),
			in.Registry.RegisterFactory("transactionInfo", &views.TransactionInfoViewFactory{}),
			in.Registry.RegisterFactory("prepareTransfer", &views.PrepareTransferViewFactory{}),
			in.Registry.RegisterFactory("broadcastPreparedTransfer", &views.BroadcastPreparedTransferViewFactory{}),
			in.Registry.RegisterFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}),
			in.Registry.RegisterFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}),
			in.Registry.RegisterFactory("SetTransactionOwnerStatus", &views.SetTransactionOwnerStatusViewFactory{}),
			in.Registry.RegisterFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}),
			in.Registry.RegisterFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}),
			in.Registry.RegisterFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}),
			in.Registry.RegisterFactory("withdrawal", &views.WithdrawalInitiatorViewFactory{}),
			in.Registry.RegisterFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}),
			in.Registry.RegisterFactory("RegisterRecipientData", &views.RegisterRecipientDataViewFactory{}),
			in.Registry.RegisterFactory("TxFinality", &views1.TxFinalityViewFactory{}),
			in.Registry.RegisterFactory("MaliciousTransfer", &views.MaliciousTransferViewFactory{}),
			in.Registry.RegisterFactory("TxStatus", &views.TxStatusViewFactory{}),
			in.Registry.RegisterFactory("TokensUpgrade", &views.TokensUpgradeInitiatorViewFactory{}),
			in.Registry.RegisterFactory("SetSpendableFlag", &views.SetSpendableFlagViewFactory{}),
			in.Registry.RegisterFactory("ListOwnerWalletIDsView", &views.ListOwnerWalletIDsViewFactory{}),
			in.Registry.RegisterFactory("RegisterOwnerIdentity", &views.RegisterOwnerIdentityViewFactory{}),
			in.Registry.RegisterFactory("TokenSelectorUnlock", &views.TokenSelectorUnlockViewFactory{}),
			in.Registry.RegisterFactory("FinalityWithTimeout", &views.FinalityWithTimeoutViewFactory{}),
			in.Registry.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{}),
			in.Registry.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{}),
			in.Registry.RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{}),
			in.Registry.RegisterResponder(&views.AcceptPreparedCashView{}, &views.PrepareTransferView{}),
			in.Registry.RegisterResponder(&views.AcceptCashView{}, &views.MultiSigLockView{}),
			in.Registry.RegisterResponder(&views.AcceptCashView{}, &views.MultiSigSpendView{}),
			in.Registry.RegisterResponder(&views.MultiSigAcceptSpendView{}, &views.MultiSigRequestSpend{}),
			in.Registry.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{}),
		)
	}); err != nil {
		return errors.WithMessage(err, "failed to install party's views")
	}
	return nil
}

func (p *SDK) Start(ctx context.Context) error {
	if p.SDK != nil {
		if err := p.SDK.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *SDK) PostStart(ctx context.Context) error {
	if p.SDK != nil {
		ps, ok := p.SDK.(node.PostStart)
		if ok {
			return ps.PostStart(ctx)
		}
	}
	return nil
}
