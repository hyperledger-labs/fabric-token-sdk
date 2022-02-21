/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/samples/fabric/fungible/views"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
)

func Topology(tokenSDKDriver string) []api.Topology {
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
	fscTopology.SetLogging("debug", "")
	fscTopology.EnableLogToFile()
	fscTopology.EnablePrometheusMetrics()

	// issuer
	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
	)
	issuer.RegisterViewFactory("issue", &views.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &views.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("register", &views.RegisterAuditorViewFactory{})

	// alice
	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
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
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
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
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "charlie.id1"),
	)
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	charlie.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	charlie.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	charlie.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	charlie.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	charlie.RegisterViewFactory("unspent", &views.ListUnspentTokensViewFactory{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetDefaultSDK(fscTopology)
	tms := tokenTopology.AddTMS(fabricTopology, tokenSDKDriver)
	tms.SetNamespace([]string{"Org1"}, "100", "2")

	// Monitoring
	//monitoringTopology := monitoring.NewTopology()
	//monitoringTopology.EnableHyperledgerExplorer()
	//monitoringTopology.EnablePrometheusGrafana()

	return []api.Topology{
		fabricTopology,
		tokenTopology,
		fscTopology,
		//monitoringTopology,
	}
}
