/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/gomega"
)

func Topology(backend string, tokenSDKDriver string, auditorAsIssuer bool) []api.Topology {
	var backendNetwork api.Topology
	backendChannel := ""
	switch backend {
	case "fabric":
		fabricTopology := fabric.NewDefaultTopology()
		fabricTopology.EnableIdemix()
		fabricTopology.AddOrganizationsByName("Org1", "Org2")
		fabricTopology.SetNamespaceApproverOrgs("Org1")
		backendNetwork = fabricTopology
		backendChannel = fabricTopology.Channels[0].Name
	case "orion":
		orionTopology := orion.NewTopology()
		backendNetwork = orionTopology
	default:
		panic("unknown backend: " + backend)
	}

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("token-sdk.core=debug:orion-sdk.rwset=debug:token-sdk.network.processor=debug:token-sdk.network.orion.custodian=debug:token-sdk.driver.identity=debug:token-sdk.driver.zkatdlog=debug:orion-sdk.vault=debug:orion-sdk.delivery=debug:orion-sdk.committer=debug:token-sdk.vault.processor=debug:info", "")
	//fscTopology.SetLogging("debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "issuer.owner"),
	)
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
	issuer.RegisterViewFactory("RegisterIssuerWallet", &views.RegisterIssuerWalletViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	issuer.RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{})
	issuer.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})

	newIssuer := fscTopology.AddNodeByName("newIssuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(),
		token.WithIssuerIdentity("newIssuer.id1"),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "newIssuer.owner"),
	)
	newIssuer.RegisterViewFactory("issue", &views.IssueCashViewFactory{})
	newIssuer.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
	newIssuer.RegisterViewFactory("GetIssuerWalletIdentity", &views.GetIssuerWalletIdentityViewFactory{})
	newIssuer.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})

	var auditor *node.Node
	newAuditor := fscTopology.AddNodeByName("newAuditor").AddOptions(
		fabric.WithOrganization("Org1"),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(),
	)
	newAuditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})
	newAuditor.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
	newAuditor.RegisterViewFactory("GetAuditorWalletIdentity", &views.GetAuditorWalletIdentityViewFactory{})
	newAuditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})

	if auditorAsIssuer {
		issuer.AddOptions(
			orion.WithRole("auditor"),
			token.WithAuditorIdentity(),
			fsc.WithAlias("auditor"),
		)
		issuer.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})
		issuer.RegisterViewFactory("historyAuditing", &views.ListAuditedTransactionsViewFactory{})
		issuer.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
		issuer.RegisterViewFactory("spending", &views.CurrentSpendingViewFactory{})
		issuer.RegisterViewFactory("SetTransactionAuditStatus", &views.SetTransactionAuditStatusViewFactory{})
		issuer.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
		issuer.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
		issuer.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
		auditor = issuer

		newIssuer.AddOptions(
			orion.WithRole("auditor"),
			token.WithAuditorIdentity(),
			fsc.WithAlias("auditor"),
		)
		newIssuer.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})
		newIssuer.RegisterViewFactory("historyAuditing", &views.ListAuditedTransactionsViewFactory{})
		newIssuer.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
		newIssuer.RegisterViewFactory("spending", &views.CurrentSpendingViewFactory{})
		newIssuer.RegisterViewFactory("SetTransactionAuditStatus", &views.SetTransactionAuditStatusViewFactory{})
		newIssuer.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
		newIssuer.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
		newIssuer.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
		newIssuer.RegisterViewFactory("GetAuditorWalletIdentity", &views.GetAuditorWalletIdentityViewFactory{})
	} else {
		auditor = fscTopology.AddNodeByName("auditor").AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			orion.WithRole("auditor"),
			token.WithAuditorIdentity(),
		)
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
	}

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("alice"),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
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

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("bob"),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
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

	charlie := fscTopology.AddNodeByName("charlie").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("charlie"),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "charlie.id1"),
	)
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{})
	charlie.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	charlie.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	charlie.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	charlie.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	charlie.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	charlie.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})
	charlie.RegisterViewFactory("balance", &views.BalanceViewFactory{})
	charlie.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	charlie.RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{})
	charlie.RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{})
	charlie.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	charlie.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	charlie.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	charlie.RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{})
	charlie.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	charlie.RegisterViewFactory("RegisterOwnerWallet", &views.RegisterOwnerWalletViewFactory{})

	manager := fscTopology.AddNodeByName("manager").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("manager"),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "manager.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "manager.id2"),
		token.WithOwnerIdentity(tokenSDKDriver, "manager.id3"),
	)
	manager.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	manager.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	manager.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	manager.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	manager.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	manager.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	manager.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	manager.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})
	manager.RegisterViewFactory("balance", &views.BalanceViewFactory{})
	manager.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	manager.RegisterViewFactory("acceptedTransactionHistory", &views.ListAcceptedTransactionsViewFactory{})
	manager.RegisterViewFactory("transactionInfo", &views.TransactionInfoViewFactory{})
	manager.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	manager.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	manager.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	manager.RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{})
	manager.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), backendNetwork, backendChannel, tokenSDKDriver)
	tms.SetNamespace("token-chaincode")
	switch tokenSDKDriver {
	case "dlog":
		// max token value is 100^2 - 1 = 9999
		tms.SetTokenGenPublicParams("100", "2")
	case "fabtoken":
		tms.SetTokenGenPublicParams("9999")
	default:
		Expect(false).To(BeTrue(), "expected token driver in (dlog,fabtoken), got [%s]", tokenSDKDriver)
	}

	fabric2.SetOrgs(tms, "Org1")
	if backend == "orion" {
		// we need to define the custodian
		custodian := fscTopology.AddNodeByName("custodian")
		custodian.AddOptions(orion.WithRole("custodian"))
		orion2.SetCustodian(tms, custodian)
		tms.AddNode(custodian)

		// Enable orion sdk on each FSC node
		orionTopology := backendNetwork.(*orion.Topology)
		orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob", "charlie", "manager")
		orionTopology.SetDefaultSDK(fscTopology)
		fscTopology.SetBootstrapNode(custodian)
	}
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms.AddAuditor(auditor)

	if backend != "orion" {
		fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))
		// Add Fabric SDK to FSC Nodes
		fscTopology.AddSDK(&fabric3.SDK{})
	}

	return []api.Topology{backendNetwork, tokenTopology, fscTopology}
}
