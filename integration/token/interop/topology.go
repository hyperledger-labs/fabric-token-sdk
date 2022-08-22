/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/htlc"
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
	//fscTopology.SetLogging("debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("history", &views.ListIssuedTokensViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("register", &views2.RegisterAuditorViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fabricTopology, fabricTopology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	return []api.Topology{fabricTopology, tokenTopology, fscTopology}
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

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("register", &views2.RegisterAuditorViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithNetworkOrganization("beta", "Org4"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	alice.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithNetworkOrganization("beta", "Org4"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	bob.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(f2Topology, f2Topology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

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

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org1"),
		fabric.WithNetworkOrganization("beta", "Org3"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("register", &views2.RegisterAuditorViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithNetworkOrganization("alpha", "Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id2"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	alice.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	alice.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	alice.RegisterViewFactory("htlc.fastExchange", &htlc.FastExchangeInitiatorViewFactory{})
	alice.RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	alice.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithNetworkOrganization("beta", "Org4"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id2"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterViewFactory("htlc.lock", &htlc.LockViewFactory{})
	bob.RegisterViewFactory("htlc.reclaimAll", &htlc.ReclaimAllViewFactory{})
	bob.RegisterViewFactory("htlc.claim", &htlc.ClaimViewFactory{})
	bob.RegisterViewFactory("htlc.scan", &htlc.ScanViewFactory{})
	bob.RegisterResponder(&htlc.LockAcceptView{}, &htlc.LockView{})
	bob.RegisterResponder(&htlc.FastExchangeResponderView{}, &htlc.FastExchangeInitiatorView{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(f1Topology, f1Topology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	tms = tokenTopology.AddTMS(f2Topology, f2Topology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org3")
	tms.AddAuditor(auditor)

	return []api.Topology{f1Topology, f2Topology, tokenTopology, fscTopology}
}
