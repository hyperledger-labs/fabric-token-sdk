/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	integration2 "github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration.AllTestTypes {
		Describe("NFT", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, t.ReplicationFactor, "alice", "bob")
			AfterEach(ts.TearDown)
			BeforeEach(ts.Setup)
			It("succeeded", func() { nft.TestAll(ts.II, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, factor int, names ...string) (*token.TestSuite, *token.ReplicaSelector) {
	opts, selector := token.NewReplicationOptions(factor, names...)
	ts := token.NewTestSuite(opts.SQLConfigs, StartPortDlog, integration2.ReplaceTemplate(nft.Topology(common.Opts{
		Backend:         "fabric",
		CommType:        commType,
		TokenSDKDriver:  "dlog",
		SDKs:            []api.SDK{&fdlog.SDK{}},
		ReplicationOpts: opts,
	})))
	return ts, selector
}
