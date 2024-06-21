/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
	_ "modernc.org/sqlite"
)

var _ = Describe("Stress EndToEnd", func() {
	Describe("T1 Fungible with Auditor ne Issuer", func() {
		ts, selector := newTestSuite()
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("stress_suite", Label("T1"), func() { fungible.TestStressSuite(ts.II, "auditor", selector) })
		It("stress", Label("T2"), func() { fungible.TestStress(ts.II, "auditor", selector) })
	})
})

func newTestSuite() (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(token2.None)
	ts := token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, topology.Topology(
		common.Opts{
			Backend:         "fabric",
			TokenSDKDriver:  "dlog",
			Aries:           true,
			ReplicationOpts: opts,
			//FSCLogSpec:     "token-sdk=debug:fabric-sdk=debug:info",
			SDKs:       []api.SDK{&fdlog.SDK{}},
			Monitoring: true,
		},
	))
	return ts, selector
}
