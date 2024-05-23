/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	views3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/htlc"
)

func HTLCSingleFabricNetworkTopology(tokenSDKDriver string, sdks ...api2.SDK) []api.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("token-sdk=debug:fabric-sdk=debug:info", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1", false),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(false),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("alice.id1"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})
	alice.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	alice.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	alice.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	alice.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	alice.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	alice.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("bob.id1"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})
	bob.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	bob.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	bob.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	bob.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	bob.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	bob.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, fabricTopology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))

	for _, sdk := range sdks {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{fabricTopology, tokenTopology, fscTopology}
}

func HTLCSingleOrionNetworkTopology(tokenSDKDriver string, sdks ...api2.SDK) []api.Topology {
	// Orion
	orionTopology := orion.NewTopology()

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithIssuerIdentity("issuer.id1", false),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(false),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("alice"),
		token.WithOwnerIdentity("alice.id1"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})
	alice.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	alice.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	alice.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	alice.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	alice.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	alice.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("bob"),
		token.WithOwnerIdentity("bob.id1"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})
	bob.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	bob.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	bob.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	bob.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	bob.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	bob.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	custodian := fscTopology.AddNodeByName("custodian")
	custodian.AddOptions(orion.WithRole("custodian"))
	fscTopology.SetBootstrapNode(custodian)

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), orionTopology, "", tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)
	orion2.SetCustodian(tms, custodian)

	orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob")

	for _, sdk := range sdks {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{orionTopology, tokenTopology, fscTopology}
}

func HTLCTwoFabricNetworksTopology(tokenSDKDriver string, sdks ...api2.SDK) []api.Topology {
	// Define two Fabric topologies
	f1Topology := fabric.NewTopologyWithName("alpha").SetDefault()
	f1Topology.EnableIdemix()
	f1Topology.AddOrganizationsByName("Org1", "Org2")
	f1Topology.SetNamespaceApproverOrgs("Org1")

	f2Topology := fabric.NewTopologyWithName("beta")
	f2Topology.EnableIdemix()
	f2Topology.AddOrganizationsByName("Org3", "Org4")
	f2Topology.SetNamespaceApproverOrgs("Org3")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("token-sdk=debug:fabric-sdk=debug:info", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1", false),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(false),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithNetworkOrganization("beta", "Org4"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("alice.id1"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimByHash", &htlc.ReclaimByHashViewFactory{})
	alice.RegisterViewFactory("htlc.CheckExistenceReceivedExpiredByHash", &htlc.CheckExistenceReceivedExpiredByHashViewFactory{})
	alice.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})
	alice.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	alice.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	alice.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	alice.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	alice.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	alice.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithNetworkOrganization("beta", "Org4"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("bob.id1"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	bob.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})
	bob.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	bob.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	bob.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	bob.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	bob.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	bob.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(fscTopology.ListNodes(), f2Topology, f2Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	for _, sdk := range sdks {
		fscTopology.AddSDK(sdk)
	}
	return []api.Topology{f1Topology, f2Topology, tokenTopology, fscTopology}
}

func HTLCNoCrossClaimTopology(tokenSDKDriver string, sdks ...api2.SDK) []api.Topology {
	// Define two Fabric topologies
	f1Topology := fabric.NewTopologyWithName("alpha").SetDefault()
	f1Topology.EnableIdemix()
	f1Topology.AddOrganizationsByName("Org1", "Org2")
	f1Topology.SetNamespaceApproverOrgs("Org1")

	f2Topology := fabric.NewTopologyWithName("beta")
	f2Topology.EnableIdemix()
	f2Topology.AddOrganizationsByName("Org3", "Org4")
	f2Topology.SetNamespaceApproverOrgs("Org3")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("db.driver.badger=info:debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1", false),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(false),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("alice.id1"),
		token.WithOwnerIdentity("alice.id2"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})
	alice.RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	alice.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	alice.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	alice.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	alice.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	alice.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	alice.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	alice.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithNetworkOrganization("beta", "Org4"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("bob.id1"),
		token.WithOwnerIdentity("bob.id2"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	bob.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})
	bob.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	bob.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	bob.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	bob.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	bob.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	bob.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "alice"), f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "bob"), f2Topology, f2Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	for _, sdk := range sdks {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{f1Topology, f2Topology, tokenTopology, fscTopology}
}

func HTLCNoCrossClaimWithOrionTopology(tokenSDKDriver string, sdks ...api2.SDK) []api.Topology {
	// Define two Fabric topologies
	f1Topology := fabric.NewTopologyWithName("alpha").SetDefault()
	f1Topology.EnableIdemix()
	f1Topology.AddOrganizationsByName("Org1", "Org2")
	f1Topology.SetNamespaceApproverOrgs("Org1")

	// Orion
	orionTopology := orion.NewTopology()
	orionTopology.SetName("beta")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("db.driver.badger=info:debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithIssuerIdentity("issuer.id1", false),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(false),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("alice.id1"),
		token.WithOwnerIdentity("alice.id2"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})
	alice.RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	alice.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	alice.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	alice.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	alice.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	alice.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	alice.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	alice.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		orion.WithRole("bob"),
		token.WithOwnerIdentity("bob.id1"),
		token.WithOwnerIdentity("bob.id2"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	bob.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})
	bob.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	bob.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	bob.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	bob.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	bob.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	bob.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})

	custodian := fscTopology.AddNodeByName("custodian")
	custodian.AddOptions(orion.WithRole("custodian"))

	tokenTopology := token.NewTopology()

	// TMS for the Fabric Network
	tmsFabric := tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "alice"), f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tmsFabric, true)
	fabric2.SetOrgs(tmsFabric, "Org1")
	tmsFabric.AddAuditor(auditor)

	// TMS for the Orion Network
	fscTopology.SetBootstrapNode(custodian)
	tmsOrion := tokenTopology.AddTMS(fscTopology.ListNodes("custodian", "auditor", "issuer", "bob"), orionTopology, "", tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tmsOrion, true)
	tmsOrion.AddAuditor(auditor)
	orion2.SetCustodian(tmsOrion, custodian)

	orionTopology.AddDB(tmsOrion.Namespace, "custodian", "issuer", "auditor", "bob")

	for _, sdk := range sdks {
		fscTopology.AddSDK(sdk)
	}
	return []api.Topology{f1Topology, orionTopology, tokenTopology, fscTopology}
}
