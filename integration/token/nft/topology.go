/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nft

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	orion3 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft/views"
)

func Topology(opts common.Opts) []api.Topology {
	var backendNetwork api.Topology
	backendChannel := ""
	switch opts.Backend {
	case "fabric":
		fabricTopology := fabric.NewDefaultTopology()
		fabricTopology.EnableIdemix()
		fabricTopology.AddOrganizationsByName("Org1", "Org2")
		fabricTopology.SetNamespaceApproverOrgs("Org1")
		backendNetwork = fabricTopology
		backendChannel = fabricTopology.Channels[0].Name
	case "orion":
		orionTopology := orion.NewTopology()
		backendNetwork = orionTopology
	default:
		panic("unknown backend: " + opts.Backend)
	}

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("grpc=error:debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(false),
	)
	issuer.RegisterViewFactory("issue", &views.IssueHouseViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(false),
	)
	auditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("alice"),
		token.WithOwnerIdentity("alice.id1"),
	)
	alice.RegisterResponder(&views.AcceptIssuedHouseView{}, &views.IssueHouseView{})
	alice.RegisterResponder(&views.AcceptTransferHouseView{}, &views.TransferHouseView{})
	alice.RegisterViewFactory("transfer", &views.TransferHouseViewFactory{})
	alice.RegisterViewFactory("queryHouse", &views.GetHouseViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("bob"),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("bob.id1"),
	)
	bob.RegisterResponder(&views.AcceptIssuedHouseView{}, &views.IssueHouseView{})
	bob.RegisterResponder(&views.AcceptTransferHouseView{}, &views.TransferHouseView{})
	bob.RegisterViewFactory("transfer", &views.TransferHouseViewFactory{})
	bob.RegisterViewFactory("queryHouse", &views.GetHouseViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	tokenTopology := token.NewTopology()
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), backendNetwork, backendChannel, opts.TokenSDKDriver)
	common.SetDefaultParams(opts.TokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	if opts.Backend == "orion" {
		// we need to define the custodian
		custodian := fscTopology.AddNodeByName("custodian")
		custodian.AddOptions(orion.WithRole("custodian"))
		orion2.SetCustodian(tms, custodian)
		tms.AddNode(custodian)

		// Enable orion sdk on each FSC node
		orionTopology := backendNetwork.(*orion.Topology)
		orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob")
		if _, ok := opts.SDKs[0].(*orion3.SDK); !ok {
			panic("orion sdk missing")
		}
		fscTopology.SetBootstrapNode(custodian)
	} else {
		if _, ok := opts.SDKs[0].(*fabric3.SDK); !ok {
			panic("fabric sdk missing")
		}
	}

	tms.AddAuditor(auditor)

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}

	return []api.Topology{backendNetwork, tokenTopology, fscTopology}
}
