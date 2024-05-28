/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken_test

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	orion3 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/ffabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fofabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/ofabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("FabToken end to end", func() {
	BeforeEach(func() {
		token.Drivers = append(token.Drivers, "fabtoken")
	})

	Describe("HTLC Single Fabric Network", func() {
		ts := token2.NewTestSuite(nil, integration2.FabTokenInteropHTLC.StartPortForNode, interop.HTLCSingleFabricNetworkTopology(common.Opts{
			CommType:        fsc.LibP2P,
			ReplicationOpts: integration.NoReplication,
			TokenSDKDriver:  "fabtoken",
			SDKs:            []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
		}))

		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Single Orion Network", func() {
		ts := token2.NewTestSuite(nil, integration2.ZKATDLogInteropHTLCOrion.StartPortForNode, interop.HTLCSingleOrionNetworkTopology(common.Opts{
			CommType:        fsc.LibP2P,
			ReplicationOpts: integration.NoReplication,
			TokenSDKDriver:  "fabtoken",
			SDKs:            []api2.SDK{&orion3.SDK{}, &ofabtoken.SDK{}},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed htlc-related basic operations", func() { interop.TestHTLCSingleNetwork(ts.II) })
	})

	Describe("HTLC Two Fabric Networks", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, integration2.FabTokenInteropHTLCTwoFabricNetworks.StartPortForNode, interop.HTLCTwoFabricNetworksTopology(common.Opts{
			CommType:        fsc.LibP2P,
			ReplicationOpts: opts,
			TokenSDKDriver:  "fabtoken",
			SDKs:            []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCTwoNetworks(ts.II, selector) })
	})

	Describe("Fast Exchange Two Fabric Networks", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, integration2.FabTokenInteropFastExchangeTwoFabricNetworks.StartPortForNode, interop.HTLCTwoFabricNetworksTopology(common.Opts{
			CommType:        fsc.LibP2P,
			ReplicationOpts: opts,
			TokenSDKDriver:  "fabtoken",
			SDKs:            []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed a fast exchange", func() { interop.TestFastExchange(ts.II, selector) })
	})

	Describe("HTLC No Cross Claim Two Fabric Networks", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, integration2.FabTokenInteropHTLCSwapNoCrossTwoFabricNetworks.StartPortForNode, interop.HTLCNoCrossClaimTopology(common.Opts{
			CommType:        fsc.LibP2P,
			ReplicationOpts: opts,
			TokenSDKDriver:  "fabtoken",
			SDKs:            []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II, selector) })
	})

	Describe("HTLC No Cross Claim with Orion and Fabric Networks", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, integration2.FabTokenInteropHTLCSwapNoCrossWithOrionAndFabricNetworks.StartPortForNode, interop.HTLCNoCrossClaimWithOrionTopology(common.Opts{
			CommType:        fsc.LibP2P,
			ReplicationOpts: opts,
			TokenSDKDriver:  "fabtoken",
			SDKs:            []api2.SDK{&fabric3.SDK{}, &orion3.SDK{}, &fofabtoken.SDK{}},
		}))
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("Performed an htlc based atomic swap", func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II, selector) })
	})

})
