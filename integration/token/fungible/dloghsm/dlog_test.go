/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dloghsm

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

const (
	Aries = 1 << iota
	AuditorAsIssuer
	NoAuditor
	HSM
	WebEnabled
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration2.AllTestTypes {
		Describe("Fungible with HSM", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|HSM, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T1"), func() { fungible.TestAll(ts.II, "auditor", nil, true, selector) })
		})

		Describe("Fungible with Auditor = Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, Aries|HSM|AuditorAsIssuer, t.ReplicationFactor, "alice", "bob", "charlie")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T2"), func() { fungible.TestAll(ts.II, "issuer", nil, true, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, mask int, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewLocalTestSuite(opts.SQLConfigs, StartPortDlog, topology.Topology(
		common.Opts{
			Backend:         "fabric",
			CommType:        commType,
			TokenSDKDriver:  "dlog",
			NoAuditor:       mask&NoAuditor > 0,
			Aries:           mask&Aries > 0,
			AuditorAsIssuer: mask&AuditorAsIssuer > 0,
			HSM:             mask&HSM > 0,
			WebEnabled:      mask&WebEnabled > 0,
			SDKs:            []api.SDK{&fdlog.SDK{}},
			// FSCLogSpec:      "token-sdk=debug:fabric-sdk=debug:info",
			ReplicationOpts: opts,
			OnlyUnity:       true,
		},
	))
	return ts, selector
}
