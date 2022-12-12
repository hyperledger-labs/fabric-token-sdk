/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken_test

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	Describe("HTLC Single Fabric Network", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropHTLC.StartPortForNode(),
				"",
				interop.HTLCSingleFabricNetworkTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed htlc-related basic operations", func() {
			interop.TestHTLCSingleNetwork(ii)
		})
	})

	Describe("HTLC Single Orion Network", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropHTLCOrion.StartPortForNode(),
				"",
				interop.HTLCSingleOrionNetworkTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed htlc-related basic operations", func() {
			interop.TestHTLCSingleNetwork(ii)
		})
	})

	Describe("HTLC Two Fabric Networks", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropHTLCTwoFabricNetworks.StartPortForNode(),
				"",
				interop.HTLCTwoFabricNetworksTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed an htlc based atomic swap", func() {
			interop.TestHTLCTwoNetworks(ii)
		})
	})

	Describe("Fast Exchange Two Fabric Networks", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropFastExchangeTwoFabricNetworks.StartPortForNode(),
				"",
				interop.HTLCTwoFabricNetworksTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed a fast exchange", func() {
			interop.TestFastExchange(ii)
		})
	})

	Describe("HTLC No Cross Claim Two Fabric Networks", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropHTLCSwapNoCrossTwoFabricNetworks.StartPortForNode(),
				"",
				interop.HTLCNoCrossClaimTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed an htlc based atomic swap", func() {
			interop.TestHTLCNoCrossClaimTwoNetworks(ii)
		})
	})

	Describe("HTLC No Cross Claim with Orion and Fabric Networks", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropHTLCSwapNoCrossWithOrionAndFabricNetworks.StartPortForNode(),
				"",
				interop.HTLCNoCrossClaimWithOrionTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed an htlc based atomic swap", func() {
			interop.TestHTLCNoCrossClaimTwoNetworks(ii)
		})
	})

	Describe("Asset Transfer With Two Fabric Networks", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.FabTokenInteropAssetTransfer.StartPortForNode(),
				"",
				interop.AssetTransferTopology("fabtoken")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed a cross network asset transfer", func() {
			interop.TestAssetTransferWithTwoNetworks(ii)
		})
	})

})
