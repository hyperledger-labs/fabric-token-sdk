/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken_test

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	orion3 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("FabToken end to end", func() {
	BeforeEach(func() {
		token.Drivers = append(token.Drivers, "fabtoken")
	})

	Describe("HTLC Single Fabric Network", func() {
		var ts = newTestSuite(fsc.LibP2P, integration2.FabTokenInteropHTLC, interop.HTLCSingleFabricNetworkTopology, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Single Orion Network", func() {
		var ts = newTestSuite(fsc.LibP2P, integration2.FabTokenInteropHTLCOrion, interop.HTLCSingleOrionNetworkTopology, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Two Fabric Networks", func() {
		var ts = newTestSuite(fsc.LibP2P, integration2.FabTokenInteropHTLCTwoFabricNetworks, interop.HTLCTwoFabricNetworksTopology, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCTwoNetworks(ts.II) })
	})

	Describe("Fast Exchange Two Fabric Networks", func() {
		var ts = newTestSuite(fsc.LibP2P, integration2.FabTokenInteropFastExchangeTwoFabricNetworks, interop.HTLCTwoFabricNetworksTopology, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed a fast exchange", func() { interop.TestFastExchange(ts.II) })
	})

	Describe("HTLC No Cross Claim Two Fabric Networks", func() {
		var ts = newTestSuite(fsc.LibP2P, integration2.FabTokenInteropHTLCSwapNoCrossTwoFabricNetworks, interop.HTLCNoCrossClaimTopology, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

	Describe("HTLC No Cross Claim with Orion and Fabric Networks", func() {
		var ts = newTestSuite(fsc.LibP2P, integration2.FabTokenInteropHTLCSwapNoCrossWithOrionAndFabricNetworks, interop.HTLCNoCrossClaimWithOrionTopology, integration.NoReplication)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})
})

func newTestSuite(commType fsc.P2PCommunicationType, portRange integration2.TestPortRange, topologies func(opts interop.Opts) []api2.Topology, opts *integration.ReplicationOptions) *token2.TestSuite {
	return token2.NewTestSuite(opts.SQLConfigs, portRange.StartPortForNode, topologies(interop.Opts{
		CommType:       commType,
		TokenSDKDriver: "fabtoken",
		FSCLogSpec:     "",
		SDKs:           []api.SDK{&orion3.SDK{}, &fabric3.SDK{}, &sdk.SDK{}},
		Replication:    opts,
	}))
}
