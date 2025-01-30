/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/odlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Stress EndToEnd", func() {
	for _, backend := range []string{
		"fabric",
		"orion",
	} {
		Describe("Stress test", Label(backend), func() {
			ts, selector := newTestSuite(backend)
			AfterEach(ts.TearDown)
			BeforeEach(ts.Setup)
			It("stress_suite", Label("T1"), func() { fungible.TestStressSuite(ts.II, "auditor", selector) })
			It("stress", Label("T2"), func() { fungible.TestStress(ts.II, "auditor", selector) })
		})
	}
})

var sdks = map[string]api.SDK{
	"fabric": &fdlog.SDK{},
	"orion":  &odlog.SDK{},
}

func newTestSuite(backend string) (*token.TestSuite, *token.ReplicaSelector) {
	// opts, selector := token.NewReplicationOptions(token.None)
	opts, selector := token.NewReplicationOptions(1, "alice", "bob", "charlie", "issuer", "auditor")
	ts := token.NewTestSuite(StartPortDlog, topology.Topology(
		common.Opts{
			Backend:         backend,
			DefaultTMSOpts:  common.TMSOpts{TokenSDKDriver: "dlog", Aries: true},
			ReplicationOpts: opts,
			CommType:        fsc.LibP2P,
			// FSCLogSpec:      "token-sdk=debug:orion-sdk=debug:info",
			SDKs:       []api.SDK{sdks[backend]},
			Monitoring: true,
		},
	))
	return ts, selector
}
