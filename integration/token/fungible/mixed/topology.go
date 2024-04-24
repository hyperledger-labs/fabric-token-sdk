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
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

const (
	DLogDriver        = "dlog"
	FabtokenDriver    = "fabtoken"
	DLogNamespace     = "dlog-token-chaincode"
	FabTokenNamespace = "fabtoken-token-chaincode"
)

func Topology() []api.Topology {
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2", "Org3")
	fabricTopology.SetNamespaceApproverOrgsOR("Org1", "Org2")
	backendNetwork := fabricTopology
	backendChannel := fabricTopology.Channels[0].Name

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("debug", "")

	issuer := fscTopology.NewTemplate("issuer")
	issuer.RegisterViewFactory("issue", &views.IssueCashViewFactory{})
	issuer.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	issuer.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	issuer.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	issuer.RegisterViewFactory("balance", &views.BalanceViewFactory{})
	issuer.RegisterViewFactory("historyIssuedToken", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("issuedTokenQuery", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	issuer.RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{})
	issuer.RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("RegisterIssuerIdentity", &views.RegisterIssuerIdentityViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	issuer.RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{})
	issuer.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	issuer1 := fscTopology.AddNodeFromTemplate("issuer1", issuer).AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
	)
	issuer2 := fscTopology.AddNodeFromTemplate("issuer2", issuer).AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
	)

	auditor := fscTopology.NewTemplate("auditor")
	auditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("historyAuditing", &views.ListAuditedTransactionsViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("spending", &views.CurrentSpendingViewFactory{})
	auditor.RegisterViewFactory("balance", &views.BalanceViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("SetTransactionAuditStatus", &views.SetTransactionAuditStatusViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
	auditor.RegisterViewFactory("GetAuditorWalletIdentity", &views.GetAuditorWalletIdentityViewFactory{})
	auditor.RegisterViewFactory("RevokeUser", &views.RevokeUserViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	auditor1 := fscTopology.AddNodeFromTemplate("auditor1", auditor).AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor2 := fscTopology.AddNodeFromTemplate("auditor2", auditor).AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("alice"),
	)
	alice.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	alice.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	alice.RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{})
	alice.RegisterResponder(&views.AcceptPreparedCashView{}, &views.PrepareTransferView{})
	alice.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	alice.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	alice.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	alice.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	alice.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})
	alice.RegisterViewFactory("balance", &views.BalanceViewFactory{})
	alice.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	alice.RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{})
	alice.RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{})
	alice.RegisterViewFactory("prepareTransfer", &views.PrepareTransferViewFactory{})
	alice.RegisterViewFactory("broadcastPreparedTransfer", &views.BroadcastPreparedTransferViewFactory{})
	alice.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	alice.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	alice.RegisterViewFactory("SetTransactionOwnerStatus", &views.SetTransactionOwnerStatusViewFactory{})
	alice.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	alice.RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{})
	alice.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("bob"),
	)
	bob.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	bob.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	bob.RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{})
	bob.RegisterResponder(&views.AcceptPreparedCashView{}, &views.PrepareTransferView{})
	bob.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	bob.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	bob.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	bob.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	bob.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	bob.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})
	bob.RegisterViewFactory("balance", &views.BalanceViewFactory{})
	bob.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	bob.RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{})
	bob.RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{})
	bob.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	bob.RegisterViewFactory("prepareTransfer", &views.PrepareTransferViewFactory{})
	bob.RegisterViewFactory("TokenSelectorUnlock", &views.TokenSelectorUnlockViewFactory{})
	bob.RegisterViewFactory("FinalityWithTimeout", &views.FinalityWithTimeoutViewFactory{})
	bob.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	bob.RegisterViewFactory("SetTransactionOwnerStatus", &views.SetTransactionOwnerStatusViewFactory{})
	bob.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	bob.RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{})
	bob.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	bob.RegisterViewFactory("GetRevocationHandle", &views.GetRevocationHandleViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

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
	fabTokenTms.SetTokenGenPublicParams("65535")
	fabric2.SetOrgs(fabTokenTms, "Org2")

	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	dlogTms.AddAuditor(auditor1)
	fabTokenTms.AddAuditor(auditor2)

	// FSC topology
	fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))
	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

	return []api.Topology{backendNetwork, tokenTopology, fscTopology}
}
