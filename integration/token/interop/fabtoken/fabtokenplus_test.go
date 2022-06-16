/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken_test

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
)

var _ = Describe("FabToken end to end", func() {
	var (
		ii *integration.Infrastructure
	)

	BeforeEach(func() {
		token.Drivers = append(token.Drivers, "fabtoken")
	})

	AfterEach(func() {
		ii.Stop()
	})

	Describe("Asset Exchange Single Fabric Network", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropExchange.StartPortForNode(),
				"",
				interop.SingleFabricNetworkTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed exchange-related basic operations", func() {
			interop.TestExchangeSingleFabricNetwork(ii)
		})
	})

})
