/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package basic

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/basic/views"
)

func Topology(tokenSDKDriver string) []nwo.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")

	// FSC
	fscTopology := fsc.NewTopology()

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "issuer.owner"),
	)
	issuer.RegisterViewFactory("issue", &views.IssueCashViewFactory{})
	issuer.RegisterViewFactory("register", &views.RegisterIssuerViewFactory{})
	issuer.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	issuer.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	issuer.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	issuer.RegisterViewFactory("history", &views.IssuerHistoryViewFactory{})
	issuer.RegisterViewFactory("issuedTokenQuery", &views.IssuerHistoryViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
	)
	auditor.RegisterViewFactory("register", &views.RegisterAuditorViewFactory{})

	certifier := fscTopology.AddNodeByName("certifier").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithCertifierIdentity(),
	)
	certifier.RegisterViewFactory("register", &views.RegisterCertifierViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
	)
	alice.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	alice.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	alice.RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{})
	alice.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	alice.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	alice.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	alice.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	alice.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
	)
	bob.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	bob.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	bob.RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{})
	bob.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	bob.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	bob.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	bob.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	bob.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	bob.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})

	charlie := fscTopology.AddNodeByName("charlie").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
	)
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.TransferWithSelectorView{})
	charlie.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	charlie.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	charlie.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	charlie.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	charlie.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	charlie.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})

	manager := fscTopology.AddNodeByName("manager").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "manager.id1"),
		token.WithOwnerIdentity(tokenSDKDriver, "manager.id2"),
		token.WithOwnerIdentity(tokenSDKDriver, "manager.id3"),
	)
	manager.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	manager.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	manager.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	manager.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	manager.RegisterViewFactory("transferWithSelector", &views.TransferWithSelectorViewFactory{})
	manager.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	manager.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	manager.RegisterViewFactory("history", &views.ListUnspentTokensViewFactory{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology)
	tms := tokenTopology.AddTMS(fabricTopology, tokenSDKDriver)
	tms.SetNamespace([]string{"Org1"}, "100", "2")
	tms.AddCertifier(certifier)

	return []nwo.Topology{fabricTopology, tokenTopology, fscTopology}
}
