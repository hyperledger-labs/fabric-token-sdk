/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	nodepkg "github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/crypto/zkatdlognoghv1"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
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
			It("succeeded", Label("T1"), func() { fungible.TestAll(context.Background(), ts.II, "auditor", nil, true, selector) })
		})

		Describe("Extras with websockets", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|WebEnabled, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("Update public params (new auditor and issuer)", Label("T2"), func() {
				fungible.TestPublicParamsUpdate(
					ts.II,
					"newAuditor",
					fungible.PrepareUpdatedPublicParams(context.Background(), ts.II, "newAuditor", "newIssuer", "default", false),
					"default",
					false,
					selector,
					false,
				)
			})
			It("Update public params (append new auditor and issuer)", Label("T2.1"), func() {
				fungible.TestPublicParamsUpdate(
					ts.II,
					"newAuditor",
					fungible.PrepareUpdatedPublicParams(context.Background(), ts.II, "newAuditor", "newIssuer", "default", true),
					"default",
					false,
					selector,
					true,
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
			It("succeeded", Label("T6"), func() { fungible.TestAll(context.Background(), ts.II, "issuer", nil, true, selector) })
			It("Update public params", Label("T7"), func() {
				fungible.TestPublicParamsUpdate(
					ts.II,
					"newIssuer",
					fungible.PrepareUpdatedPublicParams(context.Background(), ts.II, "newIssuer", "newIssuer", "default", false),
					"default",
					true,
					selector,
					false,
				)
			})
		})

		Describe("Fungible with Auditor ne Issuer + Fabric CA", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, None, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T8"), func() { fungible.TestAll(context.Background(), ts.II, "auditor", nil, false, selector) })
		})

		Describe("Malicious Transactions", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|NoAuditor, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("Malicious Transactions", Label("T9"), func() { fungible.TestMaliciousTransactions(ts.II, selector) })
		})

		Describe("Fungible with Auditor ne Issuer and Endorsers", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|WithEndorsers, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T10"), func() { fungible.TestAll(context.Background(), ts.II, "auditor", nil, true, selector) })
		})

		Describe("Multisig", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T12"), func() { fungible.TestMultiSig(ts.II, selector) })
		})

		Describe("Redeem to yourself", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries, t.ReplicationFactor, "", "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("Test redeem", Label("T13"), func() { fungible.TestRedeem(context.Background(), ts.II, selector, "default") })
		})

	}

	for _, tokenSelector := range integration2.TokenSelectors {
		Describe("TokenSelector Test", integration2.WebSocketNoReplication.Label, Label(tokenSelector), func() {
			ts, replicaSelector := newTestSuite(integration2.WebSocketNoReplication.CommType, Aries, integration2.WebSocketNoReplication.ReplicationFactor, tokenSelector, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T11"), func() { fungible.TestSelector(ts.II, "auditor", replicaSelector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, mask int, factor int, tokenSelector string, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(StartPortDlog, topology.Topology(
		common.Opts{
			Backend:             "fabric",
			CommType:            commType,
			DefaultTMSOpts:      common.TMSOpts{TokenSDKDriver: zkatdlognoghv1.DriverIdentifier, Aries: mask&Aries > 0},
			NoAuditor:           mask&NoAuditor > 0,
			AuditorAsIssuer:     mask&AuditorAsIssuer > 0,
			HSM:                 mask&HSM > 0,
			WebEnabled:          mask&WebEnabled > 0,
			SDKs:                []nodepkg.SDK{&fdlog.SDK{}},
			Monitoring:          false,
			ReplicationOpts:     opts,
			FSCBasedEndorsement: mask&WithEndorsers > 0,
			// FSCLogSpec:          "token-sdk=debug:fabric-sdk=debug:info",
			TokenSelector: tokenSelector,
		},
	))
	return ts, selector
}
