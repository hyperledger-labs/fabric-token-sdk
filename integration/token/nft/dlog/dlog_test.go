/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {

	Describe("NFT", func() {
		ts, selector := newTestSuite(fsc.LibP2P, token.None)
		AfterEach(ts.TearDown)
		BeforeEach(ts.Setup)
		It("succeeded", func() { nft.TestAll(ts.II, selector) })
	})
})

func newTestSuite(commType fsc.P2PCommunicationType, factor int, names ...string) (*token.TestSuite, *token.ReplicaSelector) {
	opts, selector := token.NewReplicationOptions(factor, names...)
	ts := token.NewTestSuite(opts.SQLConfigs, StartPortDlog, nft.Topology(common.Opts{
		Backend:         "fabric",
		CommType:        commType,
		TokenSDKDriver:  "dlog",
		SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
		ReplicationOpts: opts,
	}))
	return ts, selector
}
