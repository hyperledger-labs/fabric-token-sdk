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
	"github.com/hyperledger-labs/fabric-token-sdk/samples/nft/views"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func Orion(tokenSDKDriver string) []api.Topology {
	// Orion
	orionTopology := orion.NewTopology()

	// FSC
	fscTopology := fsc.NewTopology()
	//fscTopology.SetLogging("grpc=error:debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(),
	)
	issuer.RegisterViewFactory("issue", &views.IssueHouseViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		orion.WithRole("alice"),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
	)
	alice.RegisterResponder(&views.AcceptIssuedHouseView{}, &views.IssueHouseView{})
	alice.RegisterResponder(&views.AcceptTransferHouseView{}, &views.TransferHouseView{})
	alice.RegisterViewFactory("transfer", &views.TransferHouseViewFactory{})
	alice.RegisterViewFactory("queryHouse", &views.GetHouseViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		orion.WithRole("bob"),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
	)
	bob.RegisterResponder(&views.AcceptIssuedHouseView{}, &views.IssueHouseView{})
	bob.RegisterResponder(&views.AcceptTransferHouseView{}, &views.TransferHouseView{})
	bob.RegisterViewFactory("transfer", &views.TransferHouseViewFactory{})
	bob.RegisterViewFactory("queryHouse", &views.GetHouseViewFactory{})

	// we need to define the custodian
	custodian := fscTopology.AddNodeByName("custodian")
	custodian.AddOptions(orion.WithRole("custodian"))
	fscTopology.SetBootstrapNode(custodian)

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), orionTopology, "", tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	orion2.SetCustodian(tms, custodian)
	orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob", "charlie", "manager")
	orionTopology.SetDefaultSDK(fscTopology)
	tms.AddAuditor(auditor)

	return []api.Topology{orionTopology, tokenTopology, fscTopology}
}
