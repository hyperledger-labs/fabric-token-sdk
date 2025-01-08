/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mixed

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
)

const (
	DLogDriver        = "dlog"
	FabtokenDriver    = "fabtoken"
	DLogNamespace     = "dlog-token-chaincode"
	FabTokenNamespace = "fabtoken-token-chaincode"
)

func Topology(opts common.Opts) []api.Topology {
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2", "Org3")
	fabricTopology.SetNamespaceApproverOrgsOR("Org1", "Org2")
	backendNetwork := fabricTopology
	backendChannel := fabricTopology.Channels[0].Name

	// FSC
	fscTopology := fsc.NewTopology()
	fscTopology.P2PCommunicationType = opts.CommType
	fscTopology.SetLogging(opts.FSCLogSpec, "")

	issuer := fscTopology.NewTemplate("issuer").
		RegisterViewFactory("issue", &views.IssueCashViewFactory{}).
		RegisterViewFactory("transfer", &views.TransferViewFactory{}).
		RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{}).
		RegisterViewFactory("redeem", &views.RedeemViewFactory{}).
		RegisterViewFactory("balance", &views.BalanceViewFactory{}).
		RegisterViewFactory("historyIssuedToken", &views.ListIssuedTokensViewFactory{}).
		RegisterViewFactory("issuedTokenQuery", &views.ListIssuedTokensViewFactory{}).
		RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{}).
		RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{}).
		RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("RegisterIssuerIdentity", &views.RegisterIssuerIdentityViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}).
		RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	issuer1 := fscTopology.AddNodeFromTemplate("issuer1", issuer).
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithDefaultIssuerIdentity(false),
		).
		AddOptions(opts.ReplicationOpts.For("issuer1")...)
	issuer2 := fscTopology.AddNodeFromTemplate("issuer2", issuer).
		AddOptions(
			fabric.WithOrganization("Org2"),
			fabric.WithAnonymousIdentity(),
			token.WithDefaultIssuerIdentity(false),
		).
		AddOptions(opts.ReplicationOpts.For("issuer2")...)

	auditor := fscTopology.NewTemplate("auditor").
		RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{}).
		RegisterViewFactory("historyAuditing", &views.ListAuditedTransactionsViewFactory{}).
		RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{}).
		RegisterViewFactory("spending", &views.CurrentSpendingViewFactory{}).
		RegisterViewFactory("balance", &views.BalanceViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("SetTransactionAuditStatus", &views.SetTransactionAuditStatusViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{}).
		RegisterViewFactory("GetAuditorWalletIdentity", &views.GetAuditorWalletIdentityViewFactory{}).
		RegisterViewFactory("RevokeUser", &views.RevokeUserViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	auditor1 := fscTopology.AddNodeFromTemplate("auditor1", auditor).
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithAuditorIdentity(false),
		).
		AddOptions(opts.ReplicationOpts.For("auditor1")...)
	auditor2 := fscTopology.AddNodeFromTemplate("auditor2", auditor).
		AddOptions(
			fabric.WithOrganization("Org2"),
			fabric.WithAnonymousIdentity(),
			token.WithAuditorIdentity(false),
		).
		AddOptions(opts.ReplicationOpts.For("auditor2")...)

	alice := fscTopology.AddNodeByName("alice").
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithOwnerIdentity("alice"),
		).
		AddOptions(opts.ReplicationOpts.For("alice")...).
		RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{}).
		RegisterResponder(&views.AcceptPreparedCashView{}, &views.PrepareTransferView{}).
		RegisterViewFactory("transfer", &views.TransferViewFactory{}).
		RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{}).
		RegisterViewFactory("redeem", &views.RedeemViewFactory{}).
		RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{}).
		RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{}).
		RegisterViewFactory("balance", &views.BalanceViewFactory{}).
		RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{}).
		RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{}).
		RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{}).
		RegisterViewFactory("prepareTransfer", &views.PrepareTransferViewFactory{}).
		RegisterViewFactory("broadcastPreparedTransfer", &views.BroadcastPreparedTransferViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("SetTransactionOwnerStatus", &views.SetTransactionOwnerStatusViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{}).
		RegisterViewFactory("MaliciousTransfer", &views.MaliciousTransferViewFactory{}).
		RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{}).
		RegisterViewFactory("SetSpendableFlag", &views.SetSpendableFlagViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("bob"),
	).
		AddOptions(opts.ReplicationOpts.For("bob")...).
		RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.MaliciousTransferView{}).
		RegisterResponder(&views.AcceptPreparedCashView{}, &views.PrepareTransferView{}).
		RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{}).
		RegisterViewFactory("transfer", &views.TransferViewFactory{}).
		RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{}).
		RegisterViewFactory("redeem", &views.RedeemViewFactory{}).
		RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{}).
		RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{}).
		RegisterViewFactory("balance", &views.BalanceViewFactory{}).
		RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{}).
		RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{}).
		RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("prepareTransfer", &views.PrepareTransferViewFactory{}).
		RegisterViewFactory("TokenSelectorUnlock", &views.TokenSelectorUnlockViewFactory{}).
		RegisterViewFactory("FinalityWithTimeout", &views.FinalityWithTimeoutViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("SetTransactionOwnerStatus", &views.SetTransactionOwnerStatusViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterViewFactory("GetRevocationHandle", &views.GetRevocationHandleViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{}).
		RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{})

	// Token topology
	tokenTopology := token.NewTopology()

	// we have two TMS, one with the dlog driver and one with the fabtoken driver
	dlogTms := tokenTopology.AddTMS([]*node.Node{issuer1, auditor1, alice, bob}, backendNetwork, backendChannel, DLogDriver)
	dlogTms.SetNamespace(DLogNamespace)
	// max token value is 2^16 - 1 = 65535
	dlogTms.SetTokenGenPublicParams("16")
	fabric2.SetOrgs(dlogTms, "Org1")

	fabTokenTms := tokenTopology.AddTMS([]*node.Node{issuer2, auditor2, alice, bob}, backendNetwork, backendChannel, FabtokenDriver)
	fabTokenTms.SetNamespace(FabTokenNamespace)
	fabTokenTms.SetTokenGenPublicParams("16")
	fabric2.SetOrgs(fabTokenTms, "Org2")

	dlogTms.AddAuditor(auditor1)
	fabTokenTms.AddAuditor(auditor2)

	// FSC topology
	fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{backendNetwork, tokenTopology, fscTopology}
}
