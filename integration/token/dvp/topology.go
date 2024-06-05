/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dvp

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views/cash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp/views/house"
)

type Opts struct {
	CommType       fsc.P2PCommunicationType
	TokenSDKDriver string
	FSCLogSpec     string
	SDKs           []api2.SDK
	Replication    token2.ReplicationOpts
}

func Topology(opts Opts) []api.Topology {
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
	fscTopology.P2PCommunicationType = opts.CommType
	fscTopology.SetLogging(opts.FSCLogSpec, "")
	fscTopology.EnableLogToFile()
	fscTopology.EnablePrometheusMetrics()

	// issuer
	fscTopology.AddNodeByName("issuer").
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithDefaultIssuerIdentity(false),
		).
		AddOptions(opts.Replication.For("issuer")...).
		RegisterViewFactory("issue", &cash.IssueCashViewFactory{}).
		RegisterViewFactory("issued", &cash.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithAuditorIdentity(false),
		).
		AddOptions(opts.Replication.For("auditor")...).
		RegisterViewFactory("registerAuditor", &views2.RegisterAuditorViewFactory{})

	// issuers
	fscTopology.AddNodeByName("cash_issuer").
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithDefaultIssuerIdentity(false),
		).
		AddOptions(opts.Replication.For("cash_issuer")...).
		RegisterViewFactory("issue_cash", &cash.IssueCashViewFactory{})

	fscTopology.AddNodeByName("house_issuer").
		AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithDefaultIssuerIdentity(false),
		).
		AddOptions(opts.Replication.For("house_issuer")...).
		RegisterViewFactory("issue_house", &house.IssueHouseViewFactory{})

	fscTopology.AddNodeByName("seller").
		AddOptions(
			fabric.WithOrganization("Org2"),
			fabric.WithAnonymousIdentity(),
			token.WithDefaultOwnerIdentity(),
		).
		AddOptions(opts.Replication.For("seller")...).
		RegisterResponder(&house.AcceptHouseView{}, &house.IssueHouseView{}).
		RegisterViewFactory("sell", &views2.SellHouseViewFactory{}).
		RegisterViewFactory("queryHouse", &house.GetHouseViewFactory{}).
		RegisterViewFactory("balance", &views2.BalanceViewFactory{})

	fscTopology.AddNodeByName("buyer").
		AddOptions(
			fabric.WithOrganization("Org2"),
			fabric.WithAnonymousIdentity(),
			token.WithDefaultOwnerIdentity(),
		).
		AddOptions(opts.Replication.For("buyer")...).
		RegisterResponder(&cash.AcceptCashView{}, &cash.IssueCashView{}).
		RegisterResponder(&views2.BuyHouseView{}, &views2.SellHouseView{}).
		RegisterViewFactory("queryHouse", &house.GetHouseViewFactory{}).
		RegisterViewFactory("balance", &views2.BalanceViewFactory{}).
		RegisterViewFactory("balance", &views2.BalanceViewFactory{}).
		RegisterViewFactory("TxFinality", &views.TxFinalityViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, fabricTopology.Channels[0].Name, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	tms.AddAuditor(auditor)

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}
	return []api.Topology{
		fabricTopology,
		tokenTopology,
		fscTopology,
	}
}
