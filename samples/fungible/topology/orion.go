/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/samples/fungible/views"
)

func Orion(tokenSDKDriver string) []api.Topology {
	// Orion
	orionTopology := orion.NewTopology()

	// FSC
	fscTopology := fsc.NewTopology()
	fscTopology.SetLogging("debug", "")
	fscTopology.EnableLogToFile()
	fscTopology.EnablePrometheusMetrics()

	// issuer
	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &views2.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("register", &views2.RegisterAuditorViewFactory{})

	// alice
	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("alice"),
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
		orion.WithRole("bob"),
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
		orion.WithRole("charlie"),
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
	tokenTopology.SetDefaultSDK(fscTopology)
	tms := tokenTopology.AddTMS(orionTopology, "", tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org1")
	// we need to define the custodian
	custodian := fscTopology.AddNodeByName("custodian")
	custodian.AddOptions(orion.WithRole("custodian"))
	orion2.SetCustodian(tms, custodian)

	// Enable orion sdk on each FSC node
	orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob", "charlie")
	orionTopology.SetDefaultSDK(fscTopology)

	tokenTopology.SetDefaultSDK(fscTopology)
	tms.AddAuditor(auditor)

	// Monitoring
	//monitoringTopology := monitoring.NewTopology()
	//monitoringTopology.EnableHyperledgerExplorer()
	//monitoringTopology.EnablePrometheusGrafana()

	return []api.Topology{
		orionTopology,
		tokenTopology,
		fscTopology,
		//monitoringTopology,
	}
}
