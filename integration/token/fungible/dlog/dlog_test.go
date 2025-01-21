/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const None = 0
const (
	Aries = 1 << iota
	AuditorAsIssuer
	NoAuditor
	HSM
	WebEnabled
	WithEndorsers
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration2.AllTestTypes {
		Describe("T1 Fungible with Auditor ne Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T1"), func() { fungible.TestAll(ts.II, "auditor", nil, true, selector) })
		})

		Describe("Extras with websockets and replicas", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|WebEnabled, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("Update public params", Label("T2"), func() {
				fungible.TestPublicParamsUpdate(
					ts.II,
					"newAuditor",
					PrepareUpdatedPublicParams(ts.II, "newAuditor", "default"),
					"default",
					false,
					selector,
				)
			})
			It("Test Identity Revocation", Label("T3"), func() { fungible.TestRevokeIdentity(ts.II, "auditor", selector) })
			It("Test Remote Wallet (GRPC)", Label("T4"), func() { fungible.TestRemoteOwnerWallet(ts.II, "auditor", selector, false) })
			It("Test Remote Wallet (WebSocket)", Label("T5"), func() { fungible.TestRemoteOwnerWallet(ts.II, "auditor", selector, true) })
		})

		Describe("Fungible with Auditor = Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|AuditorAsIssuer, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T6"), func() { fungible.TestAll(ts.II, "issuer", nil, true, selector) })
			It("Update public params", Label("T7"), func() {
				fungible.TestPublicParamsUpdate(
					ts.II,
					"newIssuer",
					PrepareUpdatedPublicParams(ts.II, "newIssuer", "default"),
					"default",
					true,
					selector,
				)
			})
		})

		Describe("Fungible with Auditor ne Issuer + Fabric CA", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, None, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T8"), func() { fungible.TestAll(ts.II, "auditor", nil, false, selector) })
		})

		Describe("Malicious Transactions", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|NoAuditor, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("Malicious Transactions", Label("T9"), func() { fungible.TestMaliciousTransactions(ts.II, selector) })
		})

		Describe("T10 Fungible with Auditor ne Issuer and Endorsers", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|WithEndorsers, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T10"), func() { fungible.TestAll(ts.II, "auditor", nil, true, selector) })
		})
	}

	for _, tokenSelector := range integration2.TokenSelectors {
		Describe("T11 TokenSelector Test", integration2.WebSocketNoReplication.Label, Label(tokenSelector), func() {
			ts, replicaSelector := newTestSuite(integration2.WebSocketNoReplication.CommType, Aries, integration2.WebSocketNoReplication.ReplicationFactor, tokenSelector, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T11"), func() { fungible.TestSelector(ts.II, "auditor", replicaSelector) })
		})
	}
})

func PrepareUpdatedPublicParams(network *integration.Infrastructure, auditor string, networkName string) []byte {
	tms := fungible.GetTMSByNetworkName(network, networkName)
	auditorId := fungible.GetAuditorIdentity(tms, auditor)
	issuerId := fungible.GetIssuerIdentity(tms, "newIssuer.id1")

	tokenPlatform, ok := network.Ctx.PlatformsByName["token"].(*token.Platform)
	Expect(ok).To(BeTrue(), "failed to get token platform from context")

	// Deserialize current params
	ppBytes, err := os.ReadFile(tokenPlatform.PublicParametersFile(tms))
	Expect(err).NotTo(HaveOccurred())
	pp, err := crypto.NewPublicParamsFromBytes(ppBytes, crypto.DLogPublicParameters)
	Expect(err).NotTo(HaveOccurred())
	Expect(pp.Validate()).NotTo(HaveOccurred())

	// Update publicParameters
	pp.Auditor = auditorId
	pp.IssuerIDs = []driver.Identity{issuerId}

	// Serialize
	ppBytes, err = pp.Serialize()
	Expect(err).NotTo(HaveOccurred())

	return ppBytes
}

func newTestSuite(commType fsc.P2PCommunicationType, mask int, factor int, tokenSelector string, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, topology.Topology(
		common.Opts{
			Backend:             "fabric",
			CommType:            commType,
			DefaultTMSOpts:      common.TMSOpts{TokenSDKDriver: "dlog", Aries: mask&Aries > 0},
			NoAuditor:           mask&NoAuditor > 0,
			AuditorAsIssuer:     mask&AuditorAsIssuer > 0,
			HSM:                 mask&HSM > 0,
			WebEnabled:          mask&WebEnabled > 0,
			SDKs:                []api.SDK{&fdlog.SDK{}},
			Monitoring:          false,
			ReplicationOpts:     opts,
			FSCBasedEndorsement: mask&WithEndorsers > 0,
			// FSCLogSpec:          "token-sdk=debug:fabric-sdk=debug:info",
			OnlyUnity:     true,
			TokenSelector: tokenSelector,
		},
	))
	return ts, selector
}
