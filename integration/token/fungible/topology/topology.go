/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/monitoring"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	auditor2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/sdk/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/sdk/endorser"
	issuer2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/sdk/issuer"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/sdk/party"
)

func Topology(opts common.Opts) []api.Topology {
	if opts.Backend != "fabric" {
		panic("unknown backend: " + opts.Backend)
	}

	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")
	backendChannel := fabricTopology.Channels[0].Name

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
		token.WithDefaultIssuerIdentity(opts.HSM),
		token.WithIssuerIdentity("issuer.id1", opts.HSM),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("issuer.owner"),
	)
	issuer.AddOptions(opts.ReplicationOpts.For("issuer")...)

	newIssuer := fscTopology.AddNodeByName("newIssuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(opts.HSM),
		token.WithIssuerIdentity("newIssuer.id1", opts.HSM),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("newIssuer.owner"),
	)
	newIssuer.AddOptions(opts.ReplicationOpts.For("newIssuer")...)

	var auditor *node.Node
	if opts.AuditorAsIssuer {
		issuer.AddOptions(
			token.WithAuditorIdentity(opts.HSM),
			fsc.WithAlias("auditor"),
		)
		auditor = issuer
		newIssuer.AddOptions(
			token.WithAuditorIdentity(opts.HSM),
			fsc.WithAlias("auditor"),
		)
	} else {
		auditor = fscTopology.AddNodeByName("auditor").AddOptions(
			fabric.WithOrganization("Org1"),
			fabric.WithAnonymousIdentity(),
			token.WithAuditorIdentity(opts.HSM),
		)
		auditor.AddOptions(opts.ReplicationOpts.For("auditor")...)
	}
	newAuditor := fscTopology.AddNodeByName("newAuditor").AddOptions(
		fabric.WithOrganization("Org1"),
		token.WithAuditorIdentity(opts.HSM),
	)
	newAuditor.AddOptions(opts.ReplicationOpts.For("newAuditor")...)

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithOwnerIdentity("alice.id1"),
		token.WithRemoteOwnerIdentity("alice_remote"),
		token.WithRemoteOwnerIdentity("alice_remote_2"),
	)
	alice.AddOptions(opts.ReplicationOpts.For("alice")...)

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("bob.id1"),
		token.WithRemoteOwnerIdentity("bob_remote"),
	)
	bob.AddOptions(opts.ReplicationOpts.For("bob")...)

	charlie := fscTopology.AddNodeByName("charlie").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("charlie.id1"),
	)
	charlie.AddOptions(opts.ReplicationOpts.For("charlie")...)

	manager := fscTopology.AddNodeByName("manager").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(),
		token.WithOwnerIdentity("manager.id1"),
		token.WithOwnerIdentity("manager.id2"),
		token.WithOwnerIdentity("manager.id3"),
	)
	manager.AddOptions(opts.ReplicationOpts.For("manager")...)

	if opts.FSCBasedEndorsement {
		endorserTemplate := fscTopology.NewTemplate("endorser")
		endorserTemplate.AddOptions(
			fabric.WithOrganization("Org1"),
			fabric2.WithEndorserRole(),
		)
		fscTopology.AddNodeFromTemplate("endorser-1", endorserTemplate).AddOptions(opts.ReplicationOpts.For("endorser-1")...)
		fscTopology.AddNodeFromTemplate("endorser-2", endorserTemplate).AddOptions(opts.ReplicationOpts.For("endorser-2")...)
		fscTopology.AddNodeFromTemplate("endorser-3", endorserTemplate).AddOptions(opts.ReplicationOpts.For("endorser-3")...)
	}

	tokenTopology := token.NewTopology()
	tokenTopology.TokenSelector = opts.TokenSelector
	tms := tokenTopology.AddTMS(fscTopology.ListNodes(), fabricTopology, backendChannel, opts.DefaultTMSOpts.TokenSDKDriver)
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
	nodeList := fscTopology.ListNodes()
	fscTopology.SetBootstrapNode(fscTopology.AddNodeByName("lib-p2p-bootstrap-node"))

	if !opts.NoAuditor {
		tms.AddAuditor(auditor)
	}
	tms.AddIssuer(issuer)
	tms.AddIssuerByID("issuer.id1")

	if len(opts.SDKs) > 0 {
		// business SDKs
		// auditors
		for _, node := range fscTopology.ListNodes("auditor", "newAuditor") {
			node.AddSDKWithBase(opts.SDKs[0], &auditor2.SDK{})
		}

		// issuers
		for _, node := range fscTopology.ListNodes("issuer", "newIssuer") {
			if opts.AuditorAsIssuer {
				node.AddSDKWithBase(opts.SDKs[0], &issuer2.SDK{}, &auditor2.SDK{})
			} else {
				node.AddSDKWithBase(opts.SDKs[0], &issuer2.SDK{})
			}
		}

		// parties
		for _, node := range fscTopology.ListNodes("alice", "bob", "charlie", "manager") {
			node.AddSDKWithBase(opts.SDKs[0], &party.SDK{})
		}

		// endorsers
		if opts.FSCBasedEndorsement {
			for _, node := range fscTopology.ListNodes("endorser-1", "endorser-2", "endorser-3") {
				node.AddSDKWithBase(opts.SDKs[0], &endorser.SDK{})
			}
		}

		fscTopology.ListNodes("lib-p2p-bootstrap-node")[0].AddSDK(opts.SDKs[0])

		// add the rest of the SDKs
		for i := 1; i < len(opts.SDKs); i++ {
			fscTopology.AddSDK(opts.SDKs[i])
		}
	}

	// any extra TMS
	for _, tmsOpts := range opts.ExtraTMSs {
		tms := tokenTopology.AddTMS(nodeList, fabricTopology, backendChannel, tmsOpts.TokenSDKDriver)
		tms.Alias = tmsOpts.Alias
		tms.Namespace = "token-chaincode"
		tms.Transient = true
		if tmsOpts.Aries {
			dlog.WithAries(tms)
		}
		tms.SetTokenGenPublicParams(tmsOpts.PublicParamsGenArgs...)
		if !opts.NoAuditor {
			tms.AddAuditor(auditor)
		}
		tms.AddIssuer(issuer)
		tms.AddIssuerByID("issuer.id1")
	}

	if opts.Monitoring {
		monitoringTopology := monitoring.NewTopology()
		// monitoringTopology.EnableHyperledgerExplorer()
		monitoringTopology.EnablePrometheusGrafana()
		return []api.Topology{
			fabricTopology, tokenTopology, fscTopology,
			monitoringTopology,
		}
	}

	return []api.Topology{fabricTopology, tokenTopology, fscTopology}
}
