/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog_test

import (
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

var _ = Describe("DLog end to end", func() {
	BeforeEach(func() {
		token.Drivers = append(token.Drivers, "dlog")
	})

	Describe("HTLC Single Fabric Network with libp2p", func() {
		opts, selector := token2.NoReplication()
		ts := newTestSuite(fsc.LibP2P, integration2.ZKATDLogInteropHTLC, interop.HTLCSingleFabricNetworkTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II, selector) })
	})

	Describe("HTLC Single Fabric Network with websockets", func() {
		opts, selector := token2.NoReplication()
		ts := newTestSuite(fsc.WebSocket, integration2.ZKATDLogInteropHTLC, interop.HTLCSingleFabricNetworkTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II, selector) })
	})

	Describe("HTLC Single Fabric Network with replicas", func() {
		opts, selector := token2.NewReplicationOptions(2, "alice")
		ts := newTestSuite(fsc.WebSocket, integration2.ZKATDLogInteropHTLC, interop.HTLCSingleFabricNetworkTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II, selector) })
	})

	Describe("HTLC Single Orion Network with libp2p", func() {
		opts, selector := token2.NoReplication()
		ts := newTestSuite(fsc.LibP2P, integration2.ZKATDLogInteropHTLCOrion, interop.HTLCSingleOrionNetworkTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II, selector) })
	})

	Describe("HTLC Single Orion Network with websockets", func() {
		opts, selector := token2.NoReplication()
		ts := newTestSuite(fsc.WebSocket, integration2.ZKATDLogInteropHTLCOrion, interop.HTLCSingleOrionNetworkTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II, selector) })
	})

	Describe("HTLC Two Fabric Networks with libp2p", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, integration2.ZKATDLogInteropHTLCTwoFabricNetworks, interop.HTLCTwoFabricNetworksTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCTwoNetworks(ts.II) })
	})

	Describe("HTLC Two Fabric Networks with websockets", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.WebSocket, integration2.ZKATDLogInteropHTLCTwoFabricNetworks, interop.HTLCTwoFabricNetworksTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCTwoNetworks(ts.II) })
	})

	Describe("Fast Exchange Two Fabric Networks with libp2p", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, integration2.ZKATDLogInteropFastExchangeTwoFabricNetworks, interop.HTLCTwoFabricNetworksTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed a fast exchange", func() { interop.TestFastExchange(ts.II) })
	})

	Describe("Fast Exchange Two Fabric Networks with websockets", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.WebSocket, integration2.ZKATDLogInteropFastExchangeTwoFabricNetworks, interop.HTLCTwoFabricNetworksTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed a fast exchange", func() { interop.TestFastExchange(ts.II) })
	})

	Describe("HTLC No Cross Claim Two Fabric Networks with libp2p", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, integration2.ZKATDLogInteropHTLCSwapNoCrossTwoFabricNetworks, interop.HTLCNoCrossClaimTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

	Describe("HTLC No Cross Claim Two Fabric Networks with websockets", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.WebSocket, integration2.ZKATDLogInteropHTLCSwapNoCrossTwoFabricNetworks, interop.HTLCNoCrossClaimTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

	Describe("HTLC No Cross Claim with Orion and Fabric Networks with libp2p", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, integration2.ZKATDLogInteropHTLCSwapNoCrossWithOrionAndFabricNetworks, interop.HTLCNoCrossClaimWithOrionTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

	Describe("HTLC No Cross Claim with Orion and Fabric Networks with websockets", func() {
		opts, _ := token2.NoReplication()
		var ts = newTestSuite(fsc.WebSocket, integration2.ZKATDLogInteropHTLCSwapNoCrossWithOrionAndFabricNetworks, interop.HTLCNoCrossClaimWithOrionTopology, opts)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II) })
	})

})

func newTestSuite(commType fsc.P2PCommunicationType, portRange integration2.TestPortRange, topologies func(opts interop.Opts) []api2.Topology, opts *token2.ReplicationOptions) *token2.TestSuite {
	return token2.NewTestSuite(opts.SQLConfigs, portRange.StartPortForNode, topologies(interop.Opts{
		CommType:       commType,
		TokenSDKDriver: "dlog",
		FSCLogSpec:     "debug",
		SDKs:           []api.SDK{&orion3.SDK{}, &fabric3.SDK{}, &sdk.SDK{}},
		Replication:    opts,
	}))
}
