/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	orion3 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/ofabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Orion EndToEnd", func() {
	for _, t := range integration.AllTestTypes {
		Describe("Orion FabToken", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", func() { fungible.TestAll(ts.II, "auditor", nil, true, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, topology.Topology(
		common.Opts{
			Backend:         "orion",
			CommType:        commType,
			TokenSDKDriver:  "fabtoken",
			Aries:           true,
			SDKs:            []api.SDK{&orion3.SDK{}, &ofabtoken.SDK{}},
			ReplicationOpts: opts,
		},
	))
	return ts, selector
}
