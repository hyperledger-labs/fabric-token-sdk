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
	BeforeEach(func() { token.Drivers = append(token.Drivers, "fabtoken") })

	for _, t := range integration2.AllTestTypes {
		Describe("HTLC Single Fabric Network", t.Label, func() {
			ts, selector := newTestSuiteSingleFabric(t.CommType, t.ReplicationFactor, "alice", "bob")
			AfterEach(ts.TearDown)
			BeforeEach(ts.Setup)
			It("Performed htlc-related basic operations", Label("T1"), func() { interop.TestHTLCSingleNetwork(ts.II, selector) })
		})

		Describe("HTLC Single Orion Network", t.Label, func() {
			ts, selector := newTestSuiteSingleOrion(t.CommType, t.ReplicationFactor, "alice", "bob")
			AfterEach(ts.TearDown)
			BeforeEach(ts.Setup)
			It("Performed htlc-related basic operations", Label("T2"), func() { interop.TestHTLCSingleNetwork(ts.II, selector) })
		})

		Describe("HTLC Two Fabric Networks", t.Label, func() {
			ts, selector := newTestSuiteTwoFabric(t.CommType, t.ReplicationFactor, "alice", "bob")
			AfterEach(ts.TearDown)
			BeforeEach(ts.Setup)
			It("Performed an htlc based atomic swap", Label("T3"), func() { interop.TestHTLCTwoNetworks(ts.II, selector) })
			It("Performed a fast exchange", Label("T4"), func() { interop.TestFastExchange(ts.II, selector) })
		})

		Describe("HTLC No Cross Claim Two Fabric Networks", t.Label, func() {
			ts, selector := newTestSuiteNoCrossClaimFabric(t.CommType, t.ReplicationFactor, "alice", "bob")
			AfterEach(ts.TearDown)
			BeforeEach(ts.Setup)
			It("Performed an htlc based atomic swap", Label("T5"), func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II, selector) })
		})

		Describe("HTLC No Cross Claim with Orion and Fabric Networks", t.Label, func() {
			ts, selector := newTestSuiteNoCrossClaimOrion(t.CommType, t.ReplicationFactor)
			AfterEach(ts.TearDown)
			BeforeEach(ts.Setup)
			It("Performed an htlc based atomic swap", Label("T6"), func() { interop.TestHTLCNoCrossClaimTwoNetworks(ts.II, selector) })
		})
	}
})

func newTestSuiteSingleFabric(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, integration2.FabTokenInteropHTLC.StartPortForNode, interop.HTLCSingleFabricNetworkTopology(common.Opts{
		CommType:        commType,
		ReplicationOpts: opts,
		TokenSDKDriver:  "fabtoken",
		SDKs:            []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
	}))
	return ts, selector
}

func newTestSuiteSingleOrion(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, integration2.ZKATDLogInteropHTLCOrion.StartPortForNode, interop.HTLCSingleOrionNetworkTopology(common.Opts{
		CommType:        commType,
		ReplicationOpts: opts,
		TokenSDKDriver:  "fabtoken",
		SDKs:            []api2.SDK{&orion3.SDK{}, &ofabtoken.SDK{}},
	}))
	return ts, selector
}

func newTestSuiteTwoFabric(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, integration2.FabTokenInteropHTLCTwoFabricNetworks.StartPortForNode, interop.HTLCTwoFabricNetworksTopology(common.Opts{
		CommType:        commType,
		ReplicationOpts: opts,
		TokenSDKDriver:  "fabtoken",
		SDKs:            []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
	}))
	return ts, selector
}

func newTestSuiteNoCrossClaimFabric(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, integration2.FabTokenInteropHTLCSwapNoCrossTwoFabricNetworks.StartPortForNode, interop.HTLCNoCrossClaimTopology(common.Opts{
		CommType:        commType,
		ReplicationOpts: opts,
		TokenSDKDriver:  "fabtoken",
		SDKs:            []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
	}))
	return ts, selector
}

func newTestSuiteNoCrossClaimOrion(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, integration2.FabTokenInteropHTLCSwapNoCrossWithOrionAndFabricNetworks.StartPortForNode, interop.HTLCNoCrossClaimWithOrionTopology(common.Opts{
		CommType:        commType,
		ReplicationOpts: opts,
		TokenSDKDriver:  "fabtoken",
		SDKs:            []api2.SDK{&fabric3.SDK{}, &orion3.SDK{}, &fofabtoken.SDK{}},
	}))
	return ts, selector
}
