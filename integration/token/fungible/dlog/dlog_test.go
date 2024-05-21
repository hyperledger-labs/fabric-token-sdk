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
	fabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"
)

var _ = Describe("EndToEnd", func() {
	//Describe("T1 Fungible with Auditor ne Issuer with libp2p", func() {
	//	opts, selector := token2.NoReplication()
	//	var ts = newTestSuite(fsc.LibP2P, true, false, false, opts)
	//	BeforeEach(ts.Setup)
	//	AfterEach(ts.TearDown)
	//
	//	It("succeeded", func() {
	//		fungible.TestAll(ts.II, "auditor", nil, true)
	//	})
	//})

	//Describe("T1 Fungible with Auditor ne Issuer with websockets", func() {
	//	opts, selector := token2.NoReplication()
	//	var ts = newTestSuite(fsc.WebSocket, true, false, false, opts)
	//	BeforeEach(ts.Setup)
	//	AfterEach(ts.TearDown)
	//
	//	It("succeeded", func() {
	//		fungible.TestAll(ts.II, "auditor", nil, true)
	//	})
	//})

	Describe("T1 Fungible with Auditor ne Issuer with replicas", func() {
		opts, selector := token2.NewReplicationOptions(2, "alice")
		var ts = newTestSuite(fsc.WebSocket, true, false, false, opts)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", true, selector)
		})
	})

	Describe("Extras", func() {
		var network *integration.Infrastructure
		BeforeEach(func() {
			// notice that fabric-ca does not support yet aries
			var err error
			network, err = integration.New(StartPortDlog(), "", topology2.Topology(
				topology2.Opts{
					Backend:        "fabric",
					TokenSDKDriver: "dlog",
					Aries:          true,
					SDKs:           []api.SDK{&fabric.SDK{}, &sdk.SDK{}},
					Replication:    integration.NoReplication,
				},
			)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})
		AfterEach(func() {
			network.DeleteOnStop = false
			network.Stop()
		})

		It("Update public params", func() {
			tms := fungible.GetTMS(network, "default")
			fungible.TestPublicParamsUpdate(network, "newAuditor", PrepareUpdatedPublicParams(network, "newAuditor", tms), tms, false)
		})

		It("Test Identity Revocation", func() {
			fungible.RegisterAuditor(network, "auditor")
			rId := fungible.GetRevocationHandle(network, "bob")
			fungible.TestRevokeIdentity(network, "auditor", rId, hash.Hashable(rId).String()+" Identity is in revoked state")
		})

		It("Test Remote Wallet (GRPC)", func() {
			fungible.TestRemoteOwnerWallet(network, "auditor", false)
		})

		It("Test Remote Wallet (WebSocket)", func() {
			fungible.TestRemoteOwnerWallet(network, "auditor", true)
		})
	})

	Describe("T2 Fungible with Auditor = Issuer", func() {
		opts, selector := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, true, false, true, opts)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("T2.1 succeeded", func() {
			fungible.TestAll(ts.II, "issuer", true, selector)
		})

		It("T2.2 Update public params", func() {
			tms := fungible.GetTMS(ts.II, "default")
			fungible.TestPublicParamsUpdate(ts.II, "newIssuer", PrepareUpdatedPublicParams(ts.II, "newIssuer", tms), tms, true)
		})

	})

	Describe("T3 Fungible with Auditor ne Issuer + Fabric CA", func() {
		opts, selector := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, false, false, false, opts)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)
		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", false, selector)
		})
	})

	Describe("T4 Malicious Transactions", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, true, true, false, opts)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("Malicious Transactions", func() {
			fungible.TestMaliciousTransactions(ts.II)
		})

	})

})

func newTestSuite(commType fsc.P2PCommunicationType, aries, noAuditor, auditorAsIssuer bool, opts *token2.ReplicationOptions) *token2.TestSuite {
	return token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, topology2.Topology(
		topology2.Opts{
			Backend:         "fabric",
			CommType:        commType,
			TokenSDKDriver:  "dlog",
			Aries:           aries,
			AuditorAsIssuer: auditorAsIssuer,
			NoAuditor:       noAuditor,
			FSCLogSpec:      "debug",
			SDKs:            []api.SDK{&fabric.SDK{}, &sdk.SDK{}},
			Replication:     opts,
		},
	))
}

func PrepareUpdatedPublicParams(network *integration.Infrastructure, auditor string, tms *topology.TMS) []byte {
	auditorId := fungible.GetAuditorIdentity(network, auditor)
	issuerId := fungible.GetIssuerIdentity(network, "newIssuer.id1")
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
	pp.Issuers = [][]byte{issuerId}

	// Serialize
	ppBytes, err = pp.Serialize()
	Expect(err).NotTo(HaveOccurred())

	return ppBytes
}
