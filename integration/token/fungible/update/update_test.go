/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package update

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fall"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration.AllTestTypes {
		Describe("Updatability with Auditor ne Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, []common.TMSOpts{
				{
					Alias:               "dlog-32bits",
					TokenSDKDriver:      "dlog",
					PublicParamsGenArgs: []string{"32"},
				},
			}, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T1"), func() { fungible.TestUpdatability(ts.II, "auditor", nil, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, extraTMSs []common.TMSOpts, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, topology.Topology(common.Opts{
		Backend:         "fabric",
		CommType:        commType,
		SDKs:            []api.SDK{&fall.SDK{}},
		ReplicationOpts: opts,
		TokenSDKDriver:  "fabtoken",
		OnlyUnity:       true,
		ExtraTMSs:       extraTMSs,
		FSCLogSpec:      "token-sdk=debug:fabric-sdk=debug:info",
	}))
	return ts, selector
}
