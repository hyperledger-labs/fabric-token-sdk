/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog_test

import (
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("DLog end to end", func() {
	BeforeEach(func() {
		token.Drivers = append(token.Drivers, "dlog")
	})

	Describe("HTLC Single Fabric Network", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLC, "dlog", interop.HTLCSingleFabricNetworkTopology)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Single Orion Network", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCOrion, "dlog", interop.HTLCSingleOrionNetworkTopology)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Two Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCTwoFabricNetworks, "dlog", interop.HTLCTwoFabricNetworksTopology)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCTwoNetworks(ts.II) })
	})

	Describe("Fast Exchange Two Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropFastExchangeTwoFabricNetworks, "dlog", interop.HTLCTwoFabricNetworksTopology)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed a fast exchange", func() { interop.TestFastExchange(ts.II) })
	})

	Describe("HTLC No Cross Claim Two Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCSwapNoCrossTwoFabricNetworks, "dlog", interop.HTLCNoCrossClaimTopology)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

	Describe("HTLC No Cross Claim with Orion and Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCSwapNoCrossWithOrionAndFabricNetworks, "dlog", interop.HTLCNoCrossClaimWithOrionTopology)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

})
