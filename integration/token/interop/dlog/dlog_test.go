/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog_test

import (
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	orion3 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("DLog end to end", func() {
	BeforeEach(func() {
		token.Drivers = append(token.Drivers, "dlog")
	})

	Describe("HTLC Single Fabric Network", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLC, "dlog", interop.HTLCSingleFabricNetworkTopology, &fabric3.SDK{}, &sdk.SDK{})
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Single Orion Network", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCOrion, "dlog", interop.HTLCSingleOrionNetworkTopology, &orion3.SDK{}, &sdk.SDK{})
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Two Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCTwoFabricNetworks, "dlog", interop.HTLCTwoFabricNetworksTopology, &fabric3.SDK{}, &sdk.SDK{})
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCTwoNetworks(ts.II) })
	})

	Describe("Fast Exchange Two Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropFastExchangeTwoFabricNetworks, "dlog", interop.HTLCTwoFabricNetworksTopology, &fabric3.SDK{}, &sdk.SDK{})
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed a fast exchange", func() { interop.TestFastExchange(ts.II) })
	})

	Describe("HTLC No Cross Claim Two Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCSwapNoCrossTwoFabricNetworks, "dlog", interop.HTLCNoCrossClaimTopology, &fabric3.SDK{}, &sdk.SDK{})
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

	Describe("HTLC No Cross Claim with Orion and Fabric Networks", func() {
		var ts = interop.NewTestSuiteLibP2P(integration2.ZKATDLogInteropHTLCSwapNoCrossWithOrionAndFabricNetworks, "dlog", interop.HTLCNoCrossClaimWithOrionTopology, &orion3.SDK{}, &fabric3.SDK{}, &sdk.SDK{})
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

})
