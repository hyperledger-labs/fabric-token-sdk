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
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	views3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/htlc"
)

func HTLCSingleFabricNetworkTopology(opts common.Opts) []api.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("token-sdk=debug:fabric-sdk=debug:info", "")
	fscTopology.P2PCommunicationType = opts.CommType

	addIssuer(fscTopology).
		AddOptions(fabric.WithOrganization("Org1")).
		AddOptions(opts.ReplicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(fabric.WithOrganization("Org1")).
		AddOptions(opts.ReplicationOpts.For("auditor")...)
	addAlice(fscTopology).
		AddOptions(fabric.WithOrganization("Org2")).
		AddOptions(opts.ReplicationOpts.For("alice")...)
	addBob(fscTopology).
		AddOptions(fabric.WithOrganization("Org2")).
		AddOptions(opts.ReplicationOpts.For("bob")...)

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, fabricTopology.Channels[0].Name, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{fabricTopology, tokenTopology, fscTopology}
}

func HTLCSingleOrionNetworkTopology(opts common.Opts) []api.Topology {
	// Orion
	orionTopology := orion.NewTopology()

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("debug", "")
	fscTopology.P2PCommunicationType = opts.CommType

	addIssuer(fscTopology).
		AddOptions(fabric.WithOrganization("Org1"), orion.WithRole("issuer")).
		AddOptions(opts.ReplicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(fabric.WithOrganization("Org1"), orion.WithRole("auditor")).
		AddOptions(opts.ReplicationOpts.For("auditor")...)
	addAlice(fscTopology).
		AddOptions(fabric.WithOrganization("Org2"), orion.WithRole("alice")).
		AddOptions(opts.ReplicationOpts.For("alice")...)
	addBob(fscTopology).
		AddOptions(fabric.WithOrganization("Org2"), orion.WithRole("bob")).
		AddOptions(opts.ReplicationOpts.For("bob")...)
	custodian := addCustodian(fscTopology).
		AddOptions(opts.ReplicationOpts.For("custodian")...)

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), orionTopology, "", opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)
	orion2.SetCustodian(tms, custodian)

	orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob")

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{orionTopology, tokenTopology, fscTopology}
}

func HTLCTwoFabricNetworksTopology(opts common.Opts) []api.Topology {
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
	fscTopology.P2PCommunicationType = opts.CommType

	addIssuer(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3")).
		AddOptions(opts.ReplicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(opts.ReplicationOpts.For("auditor")...).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"))
	addAlice(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org2"),
			fabric.WithNetworkOrganization("beta", "Org4")).
		AddOptions(opts.ReplicationOpts.For("alice")...).
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{}).
		RegisterViewFactory("htlc.reclaimByHash", &htlc.ReclaimByHashViewFactory{}).
		RegisterViewFactory("htlc.CheckExistenceReceivedExpiredByHash", &htlc.CheckExistenceReceivedExpiredByHashViewFactory{}).
		RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	addBob(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org2"),
			fabric.WithNetworkOrganization("beta", "Org4"),
		).
		AddOptions(opts.ReplicationOpts.For("bob")...).
		RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{}).
		RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), f1Topology, f1Topology.Channels[0].Name, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(fscTopology.ListNodes(), f2Topology, f2Topology.Channels[0].Name, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}
	return []api.Topology{f1Topology, f2Topology, tokenTopology, fscTopology}
}

func HTLCNoCrossClaimTopology(opts common.Opts) []api.Topology {
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
	fscTopology.P2PCommunicationType = opts.CommType

	addIssuer(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"),
		).
		AddOptions(opts.ReplicationOpts.For("issuer")...)

	auditor := addAuditor(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"),
		).
		AddOptions(opts.ReplicationOpts.For("auditor")...)

	addAlice(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org2"),
			token.WithOwnerIdentity("alice.id2"),
		).
		AddOptions(opts.ReplicationOpts.For("alice")...).
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{}).
		RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})

	addBob(fscTopology).AddOptions(
		fabric.WithNetworkOrganization("beta", "Org4"),
		token.WithOwnerIdentity("bob.id2"),
	).
		AddOptions(opts.ReplicationOpts.For("bob")...).
		RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{}).
		RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "alice"), f1Topology, f1Topology.Channels[0].Name, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "bob"), f2Topology, f2Topology.Channels[0].Name, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{f1Topology, f2Topology, tokenTopology, fscTopology}
}

func HTLCNoCrossClaimWithOrionTopology(opts common.Opts) []api.Topology {
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
	fscTopology.P2PCommunicationType = opts.CommType

	addIssuer(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"),
			orion.WithRole("issuer"),
		).
		AddOptions(opts.ReplicationOpts.For("issuer")...)
	auditor := addAuditor(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org1"),
			fabric.WithNetworkOrganization("beta", "Org3"),
			orion.WithRole("auditor"),
		).
		AddOptions(opts.ReplicationOpts.For("auditor")...)
	addAlice(fscTopology).
		AddOptions(
			fabric.WithNetworkOrganization("alpha", "Org2"),
			token.WithOwnerIdentity("alice.id2"),
		).
		AddOptions(opts.ReplicationOpts.For("alice")...).
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{}).
		RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	//TODO Anonymous identity
	addBob(fscTopology).
		AddOptions(
			orion.WithRole("bob"),
			token.WithOwnerIdentity("bob.id2"),
		).
		AddOptions(opts.ReplicationOpts.For("bob")...).
		RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{}).
		RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{}).
		RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	custodian := addCustodian(fscTopology).
		AddOptions(opts.ReplicationOpts.For("custodian")...)

	tokenTopology := token.NewTopology()

	// TMS for the Fabric Network
	tmsFabric := tokenTopology.AddTMS(fscTopology.ListNodes("auditor", "issuer", "alice"), f1Topology, f1Topology.Channels[0].Name, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tmsFabric, true)
	fabric2.SetOrgs(tmsFabric, "Org1")
	tmsFabric.AddAuditor(auditor)

	// TMS for the Orion Network
	tmsOrion := tokenTopology.AddTMS(fscTopology.ListNodes("custodian", "auditor", "issuer", "bob"), orionTopology, "", opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tmsOrion, true)
	tmsOrion.AddAuditor(auditor)
	orion2.SetCustodian(tmsOrion, custodian)

	orionTopology.AddDB(tmsOrion.Namespace, "custodian", "issuer", "auditor", "bob")

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}
	return []api.Topology{f1Topology, orionTopology, tokenTopology, fscTopology}
}

func addIssuer(fscTopology *fsc.Topology) *node.Node {
	return fscTopology.AddNodeByName("issuer").
		AddOptions(
			fabric.WithAnonymousIdentity(),
			token.WithIssuerIdentity("issuer.id1", false),
			token.WithOwnerIdentity("issuer.owner"),
		).
		RegisterViewFactory("issue", &views2.IssueCashViewFactory{}).
		RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})
}

func addAuditor(fscTopology *fsc.Topology) *node.Node {
	return fscTopology.AddNodeByName("auditor").
		AddOptions(
			fabric.WithAnonymousIdentity(),
			token.WithAuditorIdentity(false),
		).
		RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{}).
		RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{}).
		RegisterViewFactory("CheckPublicParamsMatch", &views.CheckPublicParamsMatchViewFactory{}).
		RegisterViewFactory("CheckTTXDB", &views.CheckTTXDBViewFactory{}).
		RegisterViewFactory("PruneInvalidUnspentTokens", &views.PruneInvalidUnspentTokensViewFactory{}).
		RegisterViewFactory("ListVaultUnspentTokens", &views.ListVaultUnspentTokensViewFactory{}).
		RegisterViewFactory("CheckIfExistsInVault", &views.CheckIfExistsInVaultViewFactory{}).
		RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})
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
		RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{}).
		RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})
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
		RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{}).
		RegisterViewFactory("TxFinality", &views3.TxFinalityViewFactory{})
}

func addCustodian(fscTopology *fsc.Topology) *node.Node {
	custodian := fscTopology.AddNodeByName("custodian").AddOptions(orion.WithRole("custodian"))
	fscTopology.SetBootstrapNode(custodian)
	return custodian
}
