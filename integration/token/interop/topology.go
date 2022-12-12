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
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/weaver"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/htlc"
	pledge2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/pledge"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func HTLCSingleFabricNetworkTopology(tokenSDKDriver string) []api.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("db.driver.badger=info:debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})

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

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, fabricTopology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))

	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

	return []api.Topology{fabricTopology, tokenTopology, fscTopology}
}

func HTLCSingleOrionNetworkTopology(tokenSDKDriver string) []api.Topology {
	// Orion
	orionTopology := orion.NewTopology()

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})

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

	custodian := fscTopology.AddNodeByName("custodian")
	custodian.AddOptions(orion.WithRole("custodian"))
	fscTopology.SetBootstrapNode(custodian)

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), orionTopology, "", tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)
	orion2.SetCustodian(tms, custodian)

	orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob")
	orionTopology.SetDefaultSDK(fscTopology)

	return []api.Topology{orionTopology, tokenTopology, fscTopology}
}

func HTLCTwoFabricNetworksTopology(tokenSDKDriver string) []api.Topology {
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
	//fscTopology.SetLogging("debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})

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

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(fscTopology.ListNodes(), f2Topology, f2Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

	return []api.Topology{f1Topology, f2Topology, tokenTopology, fscTopology}
}

func HTLCNoCrossClaimTopology(tokenSDKDriver string) []api.Topology {
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
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})

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

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "alice"), f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "bob"), f2Topology, f2Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

	return []api.Topology{f1Topology, f2Topology, tokenTopology, fscTopology}
}

func HTLCNoCrossClaimWithOrionTopology(tokenSDKDriver string) []api.Topology {
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
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	issuer.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	issuer.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{})
	auditor.RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{})
	auditor.RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{})
	auditor.RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})

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

	custodian := fscTopology.AddNodeByName("custodian")
	custodian.AddOptions(orion.WithRole("custodian"))

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})

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
	orionTopology.SetDefaultSDK(fscTopology)

	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

	return []api.Topology{f1Topology, orionTopology, tokenTopology, fscTopology}
}

func AssetTransferTopology(tokenSDKDriver string) []api.Topology {
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

	wTopology := weaver.NewTopology()
	wTopology.AddRelayServer(f1Topology, "Org1").AddFabricNetwork(f2Topology)
	wTopology.AddRelayServer(f2Topology, "Org3").AddFabricNetwork(f1Topology)

	issuerAlpha := fscTopology.AddNodeByName("issuerAlpha").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithAnonymousIdentity(),
		fabric.WithDefaultNetwork("alpha"),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity("issuer.id1.owner"),
	)
	issuerAlpha.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuerAlpha.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	issuerAlpha.RegisterViewFactory("transfer.redeem", &pledge2.RedeemViewFactory{})
	issuerAlpha.RegisterResponder(&pledge2.IssuerResponderView{}, &pledge2.View{})
	issuerAlpha.RegisterResponder(&pledge2.IssuerResponderView{}, &pledge2.FastPledgeClaimInitiatorView{})
	issuerAlpha.RegisterResponder(&pledge2.IssuerResponderView{}, &pledge2.FastPledgeReClaimInitiatorView{})
	issuerAlpha.RegisterResponder(&pledge2.ReclaimIssuerResponderView{}, &pledge2.ReclaimInitiatorView{})
	issuerAlpha.RegisterResponder(&pledge2.ClaimIssuerView{}, &pledge2.ClaimInitiatorView{})
	issuerAlpha.RegisterResponder(&pledge2.ClaimIssuerView{}, &pledge2.FastPledgeClaimResponderView{})
	issuerAlpha.RegisterResponder(&pledge2.ClaimIssuerView{}, &pledge2.FastPledgeReClaimResponderView{})

	issuerBeta := fscTopology.AddNodeByName("issuerBeta").AddOptions(
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		fabric.WithDefaultNetwork("beta"),
		token.WithIssuerIdentity("issuer.id2"),
		token.WithOwnerIdentity("issuer.id2.owner"),
	)
	issuerBeta.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuerBeta.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	issuerBeta.RegisterViewFactory("transfer.redeem", &pledge2.RedeemViewFactory{})
	issuerBeta.RegisterResponder(&pledge2.IssuerResponderView{}, &pledge2.View{})
	issuerBeta.RegisterResponder(&pledge2.IssuerResponderView{}, &pledge2.FastPledgeClaimInitiatorView{})
	issuerBeta.RegisterResponder(&pledge2.IssuerResponderView{}, &pledge2.FastPledgeReClaimInitiatorView{})
	issuerBeta.RegisterResponder(&pledge2.ReclaimIssuerResponderView{}, &pledge2.ReclaimInitiatorView{})
	issuerBeta.RegisterResponder(&pledge2.ClaimIssuerView{}, &pledge2.ClaimInitiatorView{})
	issuerBeta.RegisterResponder(&pledge2.ClaimIssuerView{}, &pledge2.FastPledgeClaimResponderView{})
	issuerBeta.RegisterResponder(&pledge2.ClaimIssuerView{}, &pledge2.FastPledgeReClaimResponderView{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("balance", &views2.BalanceViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithAnonymousIdentity(),
		fabric.WithDefaultNetwork("alpha"),
		token.WithOwnerIdentity("alice.id1"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	alice.RegisterViewFactory("transfer.claim", &pledge2.ClaimInitiatorViewFactory{})
	alice.RegisterViewFactory("transfer.pledge", &pledge2.ViewFactory{})
	alice.RegisterViewFactory("transfer.reclaim", &pledge2.ReclaimViewFactory{})
	alice.RegisterViewFactory("transfer.fastTransfer", &pledge2.FastPledgeClaimInitiatorViewFactory{})
	alice.RegisterViewFactory("transfer.fastPledgeReclaim", &pledge2.FastPledgeReClaimInitiatorViewFactory{})
	alice.RegisterViewFactory("transfer.scan", &pledge2.ScanViewFactory{})
	alice.RegisterResponder(&pledge2.RecipientResponderView{}, &pledge2.View{})
	alice.RegisterResponder(&pledge2.FastPledgeClaimResponderView{}, &pledge2.FastPledgeClaimInitiatorView{})
	alice.RegisterResponder(&pledge2.FastPledgeReClaimResponderView{}, &pledge2.FastPledgeReClaimInitiatorView{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithNetworkOrganization("beta", "Org4"),
		fabric.WithAnonymousIdentity(),
		fabric.WithDefaultNetwork("beta"),
		token.WithOwnerIdentity("bob.id1"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterViewFactory("balance", &views2.BalanceViewFactory{})
	bob.RegisterViewFactory("transfer.claim", &pledge2.ClaimInitiatorViewFactory{})
	bob.RegisterViewFactory("transfer.pledge", &pledge2.ViewFactory{})
	bob.RegisterViewFactory("transfer.reclaim", &pledge2.ReclaimViewFactory{})
	bob.RegisterViewFactory("transfer.fastTransfer", &pledge2.FastPledgeClaimInitiatorViewFactory{})
	bob.RegisterViewFactory("transfer.fastPledgeReclaim", &pledge2.FastPledgeReClaimInitiatorViewFactory{})
	bob.RegisterViewFactory("transfer.scan", &pledge2.ScanViewFactory{})
	bob.RegisterResponder(&pledge2.RecipientResponderView{}, &pledge2.View{})
	bob.RegisterResponder(&pledge2.FastPledgeClaimResponderView{}, &pledge2.FastPledgeClaimInitiatorView{})
	bob.RegisterResponder(&pledge2.FastPledgeReClaimResponderView{}, &pledge2.FastPledgeReClaimInitiatorView{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuerAlpha", "alice"), f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuerBeta", "bob"), f2Topology, f2Topology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

	return []api.Topology{f1Topology, f2Topology, tokenTopology, wTopology, fscTopology}
}
