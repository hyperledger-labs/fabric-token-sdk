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
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"
)

var _ = Describe("EndToEnd", func() {
	Describe("T1 Fungible with Auditor ne Issuer", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, topology2.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				Aries:           true,
				SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				ReplicationOpts: opts,
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", nil, true, selector)
		})

	})

	Describe("Extras", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, topology2.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				Aries:           true,
				SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				ReplicationOpts: opts,
				WebEnabled:      true, // Needed for the Remote Wallet with websockets
			},
		))
		// notice that fabric-ca does not support yet aries
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("Update public params", func() {
			tms := fungible.GetTMS(ts.II, "default")
			fungible.TestPublicParamsUpdate(ts.II, "newAuditor", PrepareUpdatedPublicParams(ts.II, "newAuditor", tms), tms, false, selector)
		})

		It("Test Identity Revocation", func() {
			fungible.RegisterAuditor(ts.II, "auditor", nil)
			rId := fungible.GetRevocationHandle(ts.II, "bob")
			fungible.TestRevokeIdentity(ts.II, "auditor", rId, selector)
		})

		It("Test Remote Wallet (GRPC)", func() {
			fungible.TestRemoteOwnerWallet(ts.II, "auditor", selector, false)
		})

		It("Test Remote Wallet (WebSocket)", func() {
			fungible.TestRemoteOwnerWallet(ts.II, "auditor", selector, true)
		})
	})

	Describe("T2 Fungible with Auditor = Issuer", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, topology2.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				Aries:           true,
				AuditorAsIssuer: true,
				SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				ReplicationOpts: opts,
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("T2.1 succeeded", func() {
			fungible.TestAll(ts.II, "issuer", nil, true, selector)
		})

		It("T2.2 Update public params", func() {
			tms := fungible.GetTMS(ts.II, "default")
			fungible.TestPublicParamsUpdate(ts.II, "newIssuer", PrepareUpdatedPublicParams(ts.II, "newIssuer", tms), tms, true, selector)
		})

	})

	Describe("T3 Fungible with Auditor ne Issuer + Fabric CA", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, topology2.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				ReplicationOpts: opts,
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)
		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", nil, false, selector)
		})
	})

	Describe("T4 Malicious Transactions", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, topology2.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				Aries:           true,
				NoAuditor:       true,
				SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				ReplicationOpts: opts,
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("Malicious Transactions", func() {
			fungible.TestMaliciousTransactions(ts.II, selector)
		})

	})

})

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
