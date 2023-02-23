/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/samples/fungible/views"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func Orion(tokenSDKDriver string) []api.Topology {
	// Orion
	orionTopology := orion.NewTopology()

	// FSC
	fscTopology := fsc.NewTopology()
	fscTopology.SetLogging("info", "")
	fscTopology.EnableLogToFile()
	fscTopology.EnablePrometheusMetrics()

	// issuer
	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
	)
	issuer.RegisterViewFactory("issue", &views2.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &views2.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})

	// alice
	alice := fscTopology.AddNodeByName("alice").AddOptions(
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

	// we need to define the custodian
	custodian := fscTopology.AddNodeByName("custodian")
	custodian.AddOptions(orion.WithRole("custodian"))
	fscTopology.SetBootstrapNode(custodian)

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), orionTopology, "", tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	orion2.SetCustodian(tms, custodian)

	// Enable orion sdk on each FSC node
	orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob", "charlie")
	orionTopology.SetDefaultSDK(fscTopology)

	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms.AddAuditor(auditor)

	return []api.Topology{
		orionTopology,
		tokenTopology,
		fscTopology,
	}
}
