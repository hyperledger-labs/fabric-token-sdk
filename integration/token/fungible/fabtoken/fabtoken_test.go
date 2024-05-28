/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"math"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/ffabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EndToEnd", func() {
	Describe("Fungible", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, topology.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "fabtoken",
				Aries:           true,
				SDKs:            []api.SDK{&fabric.SDK{}, &ffabtoken.SDK{}},
				ReplicationOpts: opts,
				WebEnabled:      true, // Needed for the Remote Wallet with websockets
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", nil, true, selector)
		})

		It("Update public params", func() {
			network := ts.II
			auditorId := fungible.GetAuditorIdentity(network, "newAuditor")
			issuerId := fungible.GetIssuerIdentity(network, "newIssuer.id1")
			publicParam := fabtoken.PublicParams{
				Label:             "fabtoken",
				QuantityPrecision: uint64(64),
				Auditor:           auditorId,
				Issuers:           [][]byte{issuerId},
				MaxToken:          math.MaxUint64,
			}
			ppBytes, err := publicParam.Serialize()
			Expect(err).NotTo(HaveOccurred())

			tms := fungible.GetTMS(network, "default")
			Expect(tms).NotTo(BeNil())
			fungible.TestPublicParamsUpdate(network, "newAuditor", ppBytes, tms, false, selector)
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

})
