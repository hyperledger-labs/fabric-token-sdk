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
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft/views"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func Topology(backend, tokenSDKDriver string) []api.Topology {
	var backendNetwork api.Topology
	backendChannel := ""
	switch backend {
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
		panic("unknown backend: " + backend)
	}

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("grpc=error:debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(),
	)
	issuer.RegisterViewFactory("issue", &views.IssueHouseViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(),
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
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), backendNetwork, backendChannel, tokenSDKDriver)
	common.SetDefaultParams(tokenSDKDriver, tms, true)
	fabric2.SetOrgs(tms, "Org1")
	if backend == "orion" {
		// we need to define the custodian
		custodian := fscTopology.AddNodeByName("custodian")
		custodian.AddOptions(orion.WithRole("custodian"))
		orion2.SetCustodian(tms, custodian)
		tms.AddNode(custodian)

		// Enable orion sdk on each FSC node
		orionTopology := backendNetwork.(*orion.Topology)
		orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob")
		orionTopology.SetDefaultSDK(fscTopology)
		fscTopology.SetBootstrapNode(custodian)
	} else {
		// Add Fabric SDK to FSC Nodes
		fscTopology.AddSDK(&fabric3.SDK{})
	}
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms.AddAuditor(auditor)

	return []api.Topology{backendNetwork, tokenTopology, fscTopology}
}
