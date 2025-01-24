/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/odlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Orion EndToEnd", func() {
	for _, t := range integration.AllTestTypes {
		Describe("Orion ZKAT-DLog", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", func() { fungible.TestAll(ts.II, "auditor", nil, true, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(StartPortDlog, topology.Topology(
		common.Opts{
			Backend:        "orion",
			CommType:       commType,
			DefaultTMSOpts: common.TMSOpts{TokenSDKDriver: "dlog", Aries: true},
			SDKs:           []api.SDK{&odlog.SDK{}},
			// FSCLogSpec:      "token-sdk=debug:orion-sdk=debug:info",
			// FSCLogSpec:      "token-sdk=debug:orion-sdk=debug:view-sdk.services.comm=debug:info",
			ReplicationOpts: opts,
			OnlyUnity:       true,
		},
	))
	return ts, selector
}
