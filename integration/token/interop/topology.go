/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

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
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/htlc"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func HTLCSingleFabricNetworkTopology(commType fsc.P2PCommunicationType, tokenSDKDriver string, replicationOpts token2.ReplicationOpts) []api.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("token-sdk=debug:fabric-sdk=debug:info", "")
	fscTopology.P2PCommunicationType = commType

	addIssuer(fscTopology).
		AddOptions(fabric.WithOrganization("Org1")).
		AddOptions(replicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(fabric.WithOrganization("Org1")).
		AddOptions(replicationOpts.For("auditor")...)
	addAlice(fscTopology).
		AddOptions(fabric.WithOrganization("Org2")).
		AddOptions(replicationOpts.For("alice")...)
	addBob(fscTopology).
		AddOptions(fabric.WithOrganization("Org2")).
		AddOptions(replicationOpts.For("bob")...)

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

func HTLCSingleOrionNetworkTopology(commType fsc.P2PCommunicationType, tokenSDKDriver string, replicationOpts token2.ReplicationOpts) []api.Topology {
	// Orion
	orionTopology := orion.NewTopology()

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("debug", "")
	fscTopology.P2PCommunicationType = commType

	addIssuer(fscTopology).
		AddOptions(fabric.WithOrganization("Org1"), orion.WithRole("issuer")).
		AddOptions(replicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(fabric.WithOrganization("Org1"), orion.WithRole("auditor")).
		AddOptions(replicationOpts.For("auditor")...)
	addAlice(fscTopology).
		AddOptions(fabric.WithOrganization("Org2"), orion.WithRole("alice")).
		AddOptions(replicationOpts.For("alice")...)
	addBob(fscTopology).
		AddOptions(fabric.WithOrganization("Org2"), orion.WithRole("bob")).
		AddOptions(replicationOpts.For("bob")...)
	custodian := addCustodian(fscTopology).
		AddOptions(replicationOpts.For("custodian")...)

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

func HTLCTwoFabricNetworksTopology(commType fsc.P2PCommunicationType, tokenSDKDriver string, replicationOpts token2.ReplicationOpts) []api.Topology {
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
	fscTopology.P2PCommunicationType = commType

	addIssuer(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3")).
		AddOptions(replicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(replicationOpts.For("auditor")...).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"))
	addAlice(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org2"),
			fabric.WithNetworkOrganization("beta", "Org4")).
		AddOptions(replicationOpts.For("alice")...).
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{}).
		RegisterViewFactory("htlc.reclaimByHash", &htlc.ReclaimByHashViewFactory{}).
		RegisterViewFactory("htlc.CheckExistenceReceivedExpiredByHash", &htlc.CheckExistenceReceivedExpiredByHashViewFactory{}).
		RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	addBob(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org2"),
			fabric.WithNetworkOrganization("beta", "Org4"),
		).
		AddOptions(replicationOpts.For("bob")...).
		RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{}).
		RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})

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

func HTLCNoCrossClaimTopology(commType fsc.P2PCommunicationType, tokenSDKDriver string, replicationOpts token2.ReplicationOpts) []api.Topology {
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
	fscTopology.P2PCommunicationType = commType

	addIssuer(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"),
		).
		AddOptions(replicationOpts.For("issuer")...)

	auditor := addAuditor(fscTopology).AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"))

	addAlice(fscTopology).AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		token.WithOwnerIdentity("alice.id2"),
	).
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{}).
		RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})

	addBob(fscTopology).AddOptions(
		fabric.WithNetworkOrganization("beta", "Org4"),
		token.WithOwnerIdentity("bob.id2"),
	).
		RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{}).
		RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})

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

func HTLCNoCrossClaimWithOrionTopology(commType fsc.P2PCommunicationType, tokenSDKDriver string, replicationOpts token2.ReplicationOpts) []api.Topology {
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
	fscTopology.P2PCommunicationType = commType

	addIssuer(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"),
			orion.WithRole("issuer"),
		).
		AddOptions(replicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"),
			orion.WithRole("auditor"),
		).
		AddOptions(replicationOpts.For("auditor")...)
	addAlice(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org2"),
			token.WithOwnerIdentity("alice.id2"),
		).
		AddOptions(replicationOpts.For("alice")...).
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{}).
		RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	//TODO Anonymous identity
	addBob(fscTopology).
		AddOptions(
			orion.WithRole("bob"),
			token.WithOwnerIdentity("bob.id2"),
		).
		AddOptions(replicationOpts.For("bob")...).
		RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{}).
		RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	custodian := addCustodian(fscTopology).
		AddOptions(replicationOpts.For("custodian")...)

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})

	// TMS for the Fabric Network
	tmsFabric := tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "alice"), f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tmsFabric, true)
	fabric2.SetOrgs(tmsFabric, "Org1")
	tmsFabric.AddAuditor(auditor)

	// TMS for the Orion Network
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

func addIssuer(fscTopology *fsc.Topology) *node.Node {
	return fscTopology.AddNodeByName("issuer").
		AddOptions(
			fabric.WithAnonymousIdentity(),
			token.WithIssuerIdentity("issuer.id1"),
			token.WithOwnerIdentity("issuer.owner"),
		).
		RegisterViewFactory("issue", &views2.IssueCashViewFactory{}).
		RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{})
}

func addAuditor(fscTopology *fsc.Topology) *node.Node {
	return fscTopology.AddNodeByName("auditor").
		AddOptions(
			fabric.WithAnonymousIdentity(),
			token.WithAuditorIdentity(),
		).
		RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{}).
		RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{})
}

func addAlice(fscTopology *fsc.Topology) *node.Node {
	return fscTopology.AddNodeByName("alice").
		AddOptions(
			fabric.WithAnonymousIdentity(),
			token.WithOwnerIdentity("alice.id1"),
		).
		RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{}).
		RegisterViewFactory("balance", &views2.BalanceViewFactory{}).
		RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{}).
		RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{}).
		RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})
}

func addBob(fscTopology *fsc.Topology) *node.Node {
	return fscTopology.AddNodeByName("bob").
		AddOptions(
			fabric.WithAnonymousIdentity(),
			token.WithOwnerIdentity("bob.id1"),
		).
		RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{}).
		RegisterViewFactory("balance", &views2.BalanceViewFactory{}).
		RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{}).
		RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{}).
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
}

func addCustodian(fscTopology *fsc.Topology) *node.Node {
	custodian := fscTopology.AddNodeByName("custodian").AddOptions(orion.WithRole("custodian"))
	fscTopology.SetBootstrapNode(custodian)
	return custodian
}
