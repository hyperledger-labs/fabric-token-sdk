/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog_test

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DLog end to end", func() {
	var (
		ii *integration.Infrastructure
	)

	BeforeEach(func() {
		token.Drivers = append(token.Drivers, "dlog")
	})

	AfterEach(func() {
		ii.Stop()
	})

	Describe("Asset Exchange Single Fabric Network", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.ZKATDLogInteropExchange.StartPortForNode(),
				"",
				interop.AssetExchangeSingleFabricNetworkTopology("dlog")...,
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

	Describe("Asset Exchange Two Fabric Networks", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.ZKATDLogInteropExchangeTwoFabricNetworks.StartPortForNode(),
				"",
				interop.AssetExchangeTwoFabricNetworksTopology("dlog")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed an exchange based atomic swap", func() {
			interop.TestExchangeTwoFabricNetworks(ii)
		})
	})

	Describe("Fast Exchange Two Fabric Networks", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(
				integration2.ZKATDLogInteropFastExchangeTwoFabricNetworks.StartPortForNode(),
				"",
				interop.AssetExchangeTwoFabricNetworksTopology("dlog")...,
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

	Describe("Asset Exchange No Cross Claim Two Fabric Networks", func() {
		BeforeEach(func() {
			ii, err := integration.New(
				integration2.ZKATDLogInteropExchangeSwapNoCrossTwoFabricNetworks.StartPortForNode(),
				"",
				interop.AssetExchangeNoCrossClaimTopology("dlog")...,
			)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("Performed an exchange based atomic swap", func() {
			interop.TestExchangeNoCrossClaimTwoFabricNetworks(ii)
		})
	})

})
