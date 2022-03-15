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
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views/cash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views/house"
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
	)
	issuer.RegisterViewFactory("issue", &cash.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &cash.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("register", &views.RegisterAuditorViewFactory{})

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

	buyer := fscTopology.AddNodeByName("buyer").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
	)
	buyer.RegisterResponder(&cash.AcceptCashView{}, &cash.IssueCashView{})
	buyer.RegisterResponder(&views2.BuyHouseView{}, &views2.SellHouseView{})
	buyer.RegisterViewFactory("queryHouse", &house.GetHouseViewFactory{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetDefaultSDK(fscTopology)
	tms := tokenTopology.AddTMS(fabricTopology, tokenSDKDriver)
	tms.SetNamespace([]string{"Org1"}, "100", "2")

	return []api.Topology{
		fabricTopology,
		tokenTopology,
		fscTopology,
	}
}
