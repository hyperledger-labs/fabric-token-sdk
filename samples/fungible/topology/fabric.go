/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/samples/fungible/views"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func Fabric(tokenSDKDriver string) []api.Topology {
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
		token.WithDefaultIssuerIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &views2.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})

	// alice
	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
	)
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	alice.RegisterResponder(&views2.AcceptCashView{}, &views2.TransferView{})
	alice.RegisterViewFactory("transfer", &views2.TransferViewFactory{})
	alice.RegisterViewFactory("redeem", &views2.RedeemViewFactory{})
	alice.RegisterViewFactory("swap", &views2.SwapInitiatorViewFactory{})
	alice.RegisterViewFactory("unspent", &views2.ListUnspentTokensViewFactory{})

	// bob
	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
	)
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	bob.RegisterResponder(&views2.AcceptCashView{}, &views2.TransferView{})
	bob.RegisterResponder(&views2.SwapResponderView{}, &views2.SwapInitiatorView{})
	bob.RegisterViewFactory("transfer", &views2.TransferViewFactory{})
	bob.RegisterViewFactory("redeem", &views2.RedeemViewFactory{})
	bob.RegisterViewFactory("swap", &views2.SwapInitiatorViewFactory{})
	bob.RegisterViewFactory("unspent", &views2.ListUnspentTokensViewFactory{})

	// charlie
	charlie := fscTopology.AddNodeByName("charlie").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "charlie.id1"),
	)
	charlie.RegisterResponder(&views2.AcceptCashView{}, &views2.IssueCashView{})
	charlie.RegisterResponder(&views2.AcceptCashView{}, &views2.TransferView{})
	charlie.RegisterResponder(&views2.SwapResponderView{}, &views2.SwapInitiatorView{})
	charlie.RegisterViewFactory("transfer", &views2.TransferViewFactory{})
	charlie.RegisterViewFactory("redeem", &views2.RedeemViewFactory{})
	charlie.RegisterViewFactory("swap", &views2.SwapInitiatorViewFactory{})
	charlie.RegisterViewFactory("unspent", &views2.ListUnspentTokensViewFactory{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, fabricTopology.Channels[0].Name, tokenSDKDriver)
	fabric2.SetOrgs(tms, "Org1")
	tms.SetTokenGenPublicParams("100", "2")
	tms.AddAuditor(auditor)

	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

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
