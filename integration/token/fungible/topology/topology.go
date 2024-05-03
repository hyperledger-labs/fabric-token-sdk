/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	orion3 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
)

type Opts struct {
	Backend         string
	CommType        fsc.P2PCommunicationType
	TokenSDKDriver  string
	AuditorAsIssuer bool
	Aries           bool
	FSCLogSpec      string
	NoAuditor       bool
	HSM             bool
	SDKs            []api2.SDK
	Replication     token2.ReplicationOpts
}

func Topology(opts Opts) []api.Topology {
	var backendNetwork api.Topology
	backendChannel := ""
	switch opts.Backend {
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
		panic("unknown backend: " + opts.Backend)
	}

	// FSC
	fscTopology := fsc.NewTopology()
	fscTopology.P2PCommunicationType = opts.CommType
	fscTopology.SetLogging(opts.FSCLogSpec, "")

	issuer := fscTopology.AddNodeByName("issuer").
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			orion.WithRole("issuer"),
			token.WithDefaultIssuerIdentity(opts.HSM),
			token.WithIssuerIdentity("issuer.id1", opts.HSM),
			token.WithDefaultOwnerIdentity(),
			token.WithOwnerIdentity("issuer.owner"),
		).
		AddOptions(opts.Replication.For("issuer")...).
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
		RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}).
		RegisterViewFactory("SetKVSEntry", &views.SetKVSEntryViewFactory{}).
		RegisterResponder(&views.WithdrawalResponderView{}, &views.WithdrawalInitiatorView{}).
		RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	newIssuer := fscTopology.AddNodeByName("newIssuer").
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			orion.WithRole("issuer"),
			token.WithDefaultIssuerIdentity(opts.HSM),
			token.WithIssuerIdentity("newIssuer.id1", opts.HSM),
			token.WithDefaultOwnerIdentity(),
			token.WithOwnerIdentity("newIssuer.owner"),
		).
		AddOptions(opts.Replication.For("newIssuer")...).
		RegisterViewFactory("issue", &views.IssueCashViewFactory{}).
		RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}).
		RegisterViewFactory("GetIssuerWalletIdentity", &views.GetIssuerWalletIdentityViewFactory{}).
		RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{}).
		RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	fscTopology.AddNodeByName("newAuditor").
		AddOptions(
			fabric.WithOrganization("Org1"),
			orion.WithRole("auditor"),
			token.WithAuditorIdentity(opts.HSM),
		).
		AddOptions(opts.Replication.For("newAuditor")...).
		RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{}).
		RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}).
		RegisterViewFactory("GetAuditorWalletIdentity", &views.GetAuditorWalletIdentityViewFactory{}).
		RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{}).
		RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	var auditor *node.Node
	if opts.AuditorAsIssuer {
		issuer.AddOptions(
			orion.WithRole("auditor"),
			token.WithAuditorIdentity(opts.HSM),
			fsc.WithAlias("auditor"),
		).
			RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{}).
			RegisterViewFactory("historyAuditing", &views.ListAuditedTransactionsViewFactory{}).
			RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{}).
			RegisterViewFactory("spending", &views.CurrentSpendingViewFactory{}).
			RegisterViewFactory("SetTransactionAuditStatus", &views.SetTransactionAuditStatusViewFactory{}).
			RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
			RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{}).
			RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}).
			RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{})

		auditor = issuer

		newIssuer.AddOptions(
			orion.WithRole("auditor"),
			token.WithAuditorIdentity(opts.HSM),
			fsc.WithAlias("auditor"),
		).
			RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{}).
			RegisterViewFactory("historyAuditing", &views.ListAuditedTransactionsViewFactory{}).
			RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{}).
			RegisterViewFactory("spending", &views.CurrentSpendingViewFactory{}).
			RegisterViewFactory("SetTransactionAuditStatus", &views.SetTransactionAuditStatusViewFactory{}).
			RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
			RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{}).
			RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{}).
			RegisterViewFactory("GetAuditorWalletIdentity", &views.GetAuditorWalletIdentityViewFactory{})
	} else {
		auditor = fscTopology.AddNodeByName("auditor").
			AddOptions(
				fabric.WithOrganization("Org1"),
				fabric.WithAnonymousIdentity(),
				orion.WithRole("auditor"),
				token.WithAuditorIdentity(opts.HSM),
			).
			AddOptions(opts.Replication.For("auditor")...).
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
			RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}).
			RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})
	}

	addNode(fscTopology, "alice", opts.Replication).
		AddOptions(token.WithRemoteOwnerIdentity("alice_remote"),
			token.WithRemoteOwnerIdentity("alice_remote_2")).
		RegisterResponder(&views.AcceptPreparedCashView{}, &views.PrepareTransferView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{}).
		RegisterViewFactory("prepareTransfer", &views.PrepareTransferViewFactory{}).
		RegisterViewFactory("SetTransactionOwnerStatus", &views.SetTransactionOwnerStatusViewFactory{}).
		RegisterViewFactory("RegisterRecipientData", &views.RegisterRecipientDataViewFactory{}).
		RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}).
		RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{}).
		RegisterViewFactory("withdrawal", &views.WithdrawalInitiatorViewFactory{}).
		RegisterViewFactory("MaliciousTransfer", &views.MaliciousTransferViewFactory{}).
		RegisterViewFactory("broadcastPreparedTransfer", &views.BroadcastPreparedTransferViewFactory{})

	addNode(fscTopology, "bob", opts.Replication).
		AddOptions(token.WithDefaultOwnerIdentity(), token.WithRemoteOwnerIdentity("bob_remote")).
		RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.MaliciousTransferView{}).
		RegisterResponder(&views.AcceptPreparedCashView{}, &views.PrepareTransferView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{}).
		RegisterViewFactory("prepareTransfer", &views.PrepareTransferViewFactory{}).
		RegisterViewFactory("SetTransactionOwnerStatus", &views.SetTransactionOwnerStatusViewFactory{}).
		RegisterViewFactory("RegisterRecipientData", &views.RegisterRecipientDataViewFactory{}).
		RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{}).
		RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{}).
		RegisterViewFactory("TokenSelectorUnlock", &views.TokenSelectorUnlockViewFactory{}).
		RegisterViewFactory("FinalityWithTimeout", &views.FinalityWithTimeoutViewFactory{}).
		RegisterViewFactory("GetRevocationHandle", &views.GetRevocationHandleViewFactory{})

	addNode(fscTopology, "charlie", opts.Replication).
		RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{}).
		RegisterViewFactory("prepareTransfer", &views.PrepareTransferViewFactory{}).
		RegisterViewFactory("RegisterOwnerIdentity", &views.RegisterOwnerIdentityViewFactory{}).
		AddOptions(token.WithDefaultOwnerIdentity())

	addNode(fscTopology, "manager", opts.Replication).
		AddOptions(token.WithDefaultOwnerIdentity(), token.WithOwnerIdentity("manager.id2"), token.WithOwnerIdentity("manager.id3")).
		RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{}).
		RegisterViewFactory("ListOwnerWalletIDsView", &views.ListOwnerWalletIDsViewFactory{}).
		RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), backendNetwork, backendChannel, opts.TokenSDKDriver)
	tms.SetNamespace("token-chaincode")
	common.SetDefaultParams(opts.TokenSDKDriver, tms, opts.Aries)
	if !opts.Aries {
		// Enable Fabric-CA
		fabric2.WithFabricCA(tms)
	}
	fabric2.SetOrgs(tms, "Org1")
	if opts.Backend == "orion" {
		// we need to define the custodian
		custodian := fscTopology.AddNodeByName("custodian")
		custodian.AddOptions(orion.WithRole("custodian"))
		orion2.SetCustodian(tms, custodian)
		tms.AddNode(custodian)

		// Enable orion sdk on each FSC node
		orionTopology := backendNetwork.(*orion.Topology)
		orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob", "charlie", "manager")
		fscTopology.SetBootstrapNode(custodian)
		if _, ok := opts.SDKs[0].(*orion3.SDK); !ok {
			panic("orion sdk missing")
		}
	} else {
		fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))
		if _, ok := opts.SDKs[0].(*fabric3.SDK); !ok {
			panic("fabric sdk missing")
		}
	}
	if !opts.NoAuditor {
		tms.AddAuditor(auditor)
	}

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}
	return []api.Topology{backendNetwork, tokenTopology, fscTopology}
}

func addNode(fscTopology *fsc.Topology, name string, replicationOpts token2.ReplicationOpts) *node.Node {
	return fscTopology.AddNodeByName(name).
		AddOptions(
			fabric.WithOrganization("Org2"),
			fabric.WithAnonymousIdentity(),
			orion.WithRole(name),
			token.WithOwnerIdentity(fmt.Sprintf("%s.id1", name)),
		).
		AddOptions(replicationOpts.For(name)...).
		RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{}).
		RegisterResponder(&views.AcceptCashView{}, &views.TransferView{}).
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
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("WhoDeletedToken", &views.WhoDeletedTokenViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})
}
