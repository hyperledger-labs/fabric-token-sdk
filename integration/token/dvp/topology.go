/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dvp

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views/cash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views/house"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
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
	//fabricTopology.SetLogging("info", "")

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("debug", "")
	fscTopology.EnableLogToFile()
	fscTopology.EnablePrometheusMetrics()

	// issuer
	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
	)
	issuer.RegisterViewFactory("issue", &cash.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &cash.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})

	// issuers
	fscTopology.AddNodeByName("cash_issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
	).RegisterViewFactory("issue_cash", &cash.IssueCashViewFactory{})

	fscTopology.AddNodeByName("house_issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
	).RegisterViewFactory("issue_house", &house.IssueHouseViewFactory{})

	// seller and buyer
	seller := fscTopology.AddNodeByName("seller").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
	)
	seller.RegisterResponder(&house.AcceptHouseView{}, &house.IssueHouseView{})
	seller.RegisterViewFactory("sell", &views2.SellHouseViewFactory{})
	seller.RegisterViewFactory("queryHouse", &house.GetHouseViewFactory{})
	seller.RegisterViewFactory("balance", &views2.BalanceViewFactory{})

	buyer := fscTopology.AddNodeByName("buyer").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
	)
	buyer.RegisterResponder(&cash.AcceptCashView{}, &cash.IssueCashView{})
	buyer.RegisterResponder(&views2.BuyHouseView{}, &views2.SellHouseView{})
	buyer.RegisterViewFactory("queryHouse", &house.GetHouseViewFactory{})
	buyer.RegisterViewFactory("balance", &views2.BalanceViewFactory{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, fabricTopology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	// Add Fabric SDK to FSC Nodes
	fscTopology.AddSDK(&fabric3.SDK{})

	return []api.Topology{
		fabricTopology,
		tokenTopology,
		fscTopology,
	}
}
