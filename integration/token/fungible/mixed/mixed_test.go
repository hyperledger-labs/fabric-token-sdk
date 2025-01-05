/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mixed

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fall"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration.AllTestTypes {
		Describe("Fungible with Auditor ne Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, nil, t.ReplicationFactor, "alice", "bob")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T1"), func() { fungible.TestMixed(ts.II, nil, selector) })
		})
		Describe("Updatability with Auditor ne Issuer", t.Label, func() {
			ts, selector := newTestSuite(
				t.CommType,
				make([]common.TMSOpts, 0),
				t.ReplicationFactor,
				"alice",
				"bob",
			)
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("upgrade", Label("T2"), func() {
				fungible.TestUpdatability(ts.II, nil, selector)
			})
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, extraTMSs []common.TMSOpts, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, Topology(common.Opts{
		CommType:        commType,
		SDKs:            []api.SDK{&fall.SDK{}},
		ReplicationOpts: opts,
		ExtraTMSs:       extraTMSs,
	}))
	return ts, selector
}
