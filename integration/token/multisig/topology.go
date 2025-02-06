/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/monitoring"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/common"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	views2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/views"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
	views3 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/multisig/views"
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
	fscTopology.P2PCommunicationType = opts.CommType
	fscTopology.WebEnabled = opts.WebEnabled
	if opts.Monitoring {
		fscTopology.EnablePrometheusMetrics()
		fscTopology.EnableTracing(tracing.File)
	}
	fscTopology.SetLogging(opts.FSCLogSpec, "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("issuer"),
		token.WithDefaultIssuerIdentity(opts.HSM),
		token.WithIssuerIdentity("issuer.id1", opts.HSM),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.AddOptions(opts.ReplicationOpts.For("issuer")...)
	issuer.RegisterViewFactory("issue", &views.IssueCashViewFactory{})
	issuer.RegisterViewFactory("balance", &views.BalanceViewFactory{})
	issuer.RegisterViewFactory("historyIssuedToken", &views.ListIssuedTokensViewFactory{})
	issuer.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})
	issuer.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
	issuer.RegisterViewFactory("GetIssuerWalletIdentity", &views.GetIssuerWalletIdentityViewFactory{})
	issuer.RegisterViewFactory("DoesWalletExist", &views.DoesWalletExistViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("auditor"),
		token.WithAuditorIdentity(opts.HSM),
	)
	auditor.AddOptions(opts.ReplicationOpts.For("auditor")...)
	auditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})
	auditor.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
	auditor.RegisterViewFactory("GetAuditorWalletIdentity", &views.GetAuditorWalletIdentityViewFactory{})
	auditor.RegisterViewFactory("holding", &views.CurrentHoldingViewFactory{})
	auditor.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("alice"),
		token.WithOwnerIdentity("alice.id1"),
		token.WithRemoteOwnerIdentity("alice_remote"),
	)
	alice.AddOptions(opts.ReplicationOpts.For("alice")...)
	alice.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	alice.RegisterResponder(&views3.AcceptCashView{}, &views3.LockView{})
	alice.RegisterResponder(&views3.AcceptCashView{}, &views3.LockWithSelectorView{})
	alice.RegisterViewFactory("lock", &views3.LockViewFactory{})
	alice.RegisterViewFactory("lockWithSelector", &views3.LockWithSelectorViewFactory{})
	alice.RegisterViewFactory("balance", &views3.BalanceViewFactory{})
	alice.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	alice.RegisterViewFactory("RegisterRecipientData", &views.RegisterRecipientDataViewFactory{})
	alice.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})
	alice.RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("bob"),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("bob.id1"),
		token.WithRemoteOwnerIdentity("bob_remote"),
	)
	bob.AddOptions(opts.ReplicationOpts.For("bob")...)
	bob.RegisterResponder(&views3.AcceptCashView{}, &views3.LockView{})
	bob.RegisterResponder(&views3.AcceptCashView{}, &views3.LockWithSelectorView{})
	bob.RegisterViewFactory("lock", &views3.LockViewFactory{})
	bob.RegisterViewFactory("lockWithSelector", &views3.LockWithSelectorViewFactory{})
	bob.RegisterViewFactory("balance", &views3.BalanceViewFactory{})
	bob.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	bob.RegisterViewFactory("RegisterRecipientData", &views.RegisterRecipientDataViewFactory{})
	bob.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})
	bob.RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{})

	charlie := fscTopology.AddNodeByName("charlie").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("charlie"),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("charlie.id1"),
	)
	charlie.AddOptions(opts.ReplicationOpts.For("charlie")...)
	charlie.RegisterResponder(&views3.AcceptCashView{}, &views3.LockView{})
	charlie.RegisterResponder(&views3.AcceptCashView{}, &views3.LockWithSelectorView{})
	charlie.RegisterViewFactory("lock", &views3.LockViewFactory{})
	charlie.RegisterViewFactory("lockWithSelector", &views3.LockWithSelectorViewFactory{})
	charlie.RegisterViewFactory("balance", &views3.BalanceViewFactory{})
	charlie.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	charlie.RegisterViewFactory("RegisterRecipientData", &views.RegisterRecipientDataViewFactory{})
	charlie.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})
	charlie.RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{})

	dave := fscTopology.AddNodeByName("dave").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		orion.WithRole("dave"),
		token.WithOwnerIdentity("dave.id1"),
		token.WithRemoteOwnerIdentity("dasve_remote"),
	)
	dave.AddOptions(opts.ReplicationOpts.For("dave")...)
	dave.RegisterResponder(&views3.AcceptCashView{}, &views3.LockView{})
	dave.RegisterResponder(&views3.AcceptCashView{}, &views3.LockWithSelectorView{})
	dave.RegisterViewFactory("lock", &views3.LockViewFactory{})
	dave.RegisterViewFactory("lockWithSelector", &views3.LockWithSelectorViewFactory{})
	dave.RegisterViewFactory("balance", &views3.BalanceViewFactory{})
	dave.RegisterViewFactory("GetEnrollmentID", &views.GetEnrollmentIDViewFactory{})
	dave.RegisterViewFactory("RegisterRecipientData", &views.RegisterRecipientDataViewFactory{})
	dave.RegisterViewFactory("TxFinality", &views2.TxFinalityViewFactory{})
	dave.RegisterViewFactory("TxStatus", &views.TxStatusViewFactory{})

	if opts.FSCBasedEndorsement {
		endorserTemplate := fscTopology.NewTemplate("endorser")
		endorserTemplate.RegisterViewFactory("GetPublicParams", &views.GetPublicParamsViewFactory{})
		endorserTemplate.RegisterViewFactory("EndorserFinality", &endorser.FinalityViewFactory{})
		endorserTemplate.AddOptions(
			fabric.WithOrganization("Org1"),
			fabric2.WithEndorserRole(),
		)
		fscTopology.AddNodeFromTemplate("endorser-1", endorserTemplate)
		fscTopology.AddNodeFromTemplate("endorser-2", endorserTemplate)
		fscTopology.AddNodeFromTemplate("endorser-3", endorserTemplate)
	}

	tokenTopology := token.NewTopology()
	tokenTopology.TokenSelector = opts.TokenSelector
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), backendNetwork, backendChannel, opts.DefaultTMSOpts.TokenSDKDriver)
	tms.SetNamespace("token-chaincode")
	common.SetDefaultParams(tms, opts.DefaultTMSOpts)
	if !opts.DefaultTMSOpts.Aries {
		// Enable Fabric-CA
		fabric2.WithFabricCA(tms)
	}
	if opts.FSCBasedEndorsement {
		fabric2.WithFSCEndorsers(tms, "endorser-1", "endorser-2", "endorser-3")
	}
	fabric2.SetOrgs(tms, "Org1")
	var nodeList []*node.Node
	if opts.Backend == "orion" {
		// we need to define the custodian
		custodian := fscTopology.AddNodeByName("custodian")
		custodian.AddOptions(orion.WithRole("custodian"))
		custodian.AddOptions(opts.ReplicationOpts.For("custodian")...)
		orion2.SetCustodian(tms, custodian.Name)
		tms.AddNode(custodian)

		// Enable orion sdk on each FSC node
		orionTopology := backendNetwork.(*orion.Topology)
		orionTopology.AddDB(tms.Namespace, "custodian", "issuer", "auditor", "alice", "bob", "charlie", "manager")
		fscTopology.SetBootstrapNode(custodian)
		nodeList = fscTopology.ListNodes()
	} else {
		nodeList = fscTopology.ListNodes()
		fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))
	}

	if !opts.NoAuditor {
		tms.AddAuditor(auditor)
	}

	if opts.OnlyUnity {
		common2.WithOnlyUnity(tms)
	}

	for _, sdk := range opts.SDKs {
		fscTopology.AddSDK(sdk)
	}

	// any extra TMS
	for _, tmsOpts := range opts.ExtraTMSs {
		tms := tokenTopology.AddTMS(nodeList, backendNetwork, backendChannel, tmsOpts.TokenSDKDriver)
		tms.Alias = tmsOpts.Alias
		tms.Namespace = "token-chaincode"
		tms.Transient = true
		if tmsOpts.Aries {
			dlog.WithAries(tms)
		}
		tms.SetTokenGenPublicParams(tmsOpts.PublicParamsGenArgs...)
	}

	if opts.Monitoring {
		monitoringTopology := monitoring.NewTopology()
		// monitoringTopology.EnableHyperledgerExplorer()
		monitoringTopology.EnablePrometheusGrafana()
		return []api.Topology{
			backendNetwork, tokenTopology, fscTopology,
			monitoringTopology,
		}
	}

	return []api.Topology{backendNetwork, tokenTopology, fscTopology}
}
