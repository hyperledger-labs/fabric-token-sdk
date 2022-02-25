/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package dvp

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views"
)

func Topology(tokenSDKDriver string) []api.Topology {
	orgs := []string{"TokenOrg", "HouseOrg", "Org1"}

	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName(orgs...)
	fabricTopology.SetNamespaceApproverOrgsOR("TokenOrg", "HouseOrg")
	fabricTopology.AddNamespaceWithUnanimity("house", "HouseOrg").SetStateChaincode()

	fscTopology := fsc.NewTopology()

	// approver - "TokenOrg", "HouseOrg"
	approver := fscTopology.AddNodeByName("token_approver").AddOptions(fabric.WithOrganization("TokenOrg"))
	approver.RegisterResponder(&views.TokenApproveView{}, &views.IssueCashView{})
	approver.RegisterResponder(&views.TokenApproveView{}, &views.SellHouseView{})

	approver = fscTopology.AddNodeByName("house_approver").AddOptions(fabric.WithOrganization("HouseOrg"))
	approver.RegisterResponder(&views.HouseApproveView{}, &views.IssueHouseView{})
	approver.RegisterResponder(&views.HouseApproveView{}, &views.SellHouseView{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("TokenOrg"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("register", &views.RegisterAuditorViewFactory{})

	// issuers
	fscTopology.AddNodeByName("cash_issuer").AddOptions(
		fabric.WithOrganization("TokenOrg"),
		fabric.WithAnonymousIdentity(),
		token.WithIssuerIdentity("cash_issuer"),
	).RegisterViewFactory("issue_cash", &views.IssueCashViewFactory{})

	fscTopology.AddNodeByName("house_issuer").AddOptions(
		fabric.WithOrganization("HouseOrg"),
		fabric.WithAnonymousIdentity(),
	).RegisterViewFactory("issue_house", &views.IssueHouseViewFactory{})

	// seller and buyer
	seller := fscTopology.AddNodeByName("seller").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "seller.id1"),
	)
	seller.RegisterResponder(&views.AcceptHouseView{}, &views.IssueHouseView{})
	seller.RegisterViewFactory("sell", &views.SellHouseViewFactory{})

	buyer := fscTopology.AddNodeByName("buyer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity(tokenSDKDriver, "buyer.id1"),
	)
	buyer.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	buyer.RegisterResponder(&views.BuyHouseView{}, &views.SellHouseView{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetDefaultSDK(fscTopology)
	tms := tokenTopology.AddTMS(fabricTopology, tokenSDKDriver)
	tms.SetNamespace([]string{"TokenOrg"}, "100", "1")
	tms.AddAuditor(auditor)

	return []api.Topology{fabricTopology, tokenTopology, fscTopology}
}
