/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"math"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EndToEnd", func() {
	var (
		network *integration.Infrastructure
	)

	AfterEach(func() {
		network.Stop()
	})

	Describe("Fungible", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", topology.Topology("fabric", "fabtoken", false)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "auditor", nil)
		})

		It("Update public params", func() {
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
			fungible.TestPublicParamsUpdate(network, "newAuditor", ppBytes, tms, false)
		})
	})

})
