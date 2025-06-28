/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/artifactgen"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
)

func Topology(tokenSDKDriver string, sdks ...node.SDK) []api.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")
	fabricTopology.EnableGRPCLogging()
	fabricTopology.EnableLogPeersToFile()
	fabricTopology.EnableLogOrderersToFile()
	fabricTopology.SetLogging("info", "")

	// FSC
	fscTopology := fsc.NewTopology()
	fscTopology.SetLogging("info", "")
	fscTopology.EnableLogToFile()
	fscTopology.EnablePrometheusMetrics()

	// issuer
	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(false),
		token.WithIssuerIdentity("issuer.id1", false),
	)
	issuer.RegisterViewFactory("issue", &views.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &views.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(false),
	)
	auditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})

	// alice
	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("alice.id1"),
	)
	alice.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	alice.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	alice.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	alice.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	alice.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	alice.RegisterViewFactory("unspent", &views.ListUnspentTokensViewFactory{})

	// bob
	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("bob.id1"),
	)
	bob.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	bob.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	bob.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	bob.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	bob.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	bob.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	bob.RegisterViewFactory("unspent", &views.ListUnspentTokensViewFactory{})

	// charlie
	charlie := fscTopology.AddNodeByName("charlie").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("charlie.id1"),
	)
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	charlie.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	charlie.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	charlie.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	charlie.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	charlie.RegisterViewFactory("unspent", &views.ListUnspentTokensViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, fabricTopology.Channels[0].Name, tokenSDKDriver)
	fabric2.SetOrgs(tms, "Org1")
	tms.SetTokenGenPublicParams("16")
	tms.AddAuditor(auditor)
	tms.AddIssuer(issuer)

	for _, sdk := range sdks {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{
		fabricTopology,
		tokenTopology,
		fscTopology,
	}
}

func main() {
	if err := artifactgen.WriteTopologies("fungible.yaml", Topology("dlog", &tokensdk.SDK{}), 0766); err != nil {
		panic(err)
	}
}
