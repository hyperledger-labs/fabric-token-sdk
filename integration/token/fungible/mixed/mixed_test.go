/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mixed

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	nodepkg "github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fall"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration.WebSocketNoReplicationOnly {
		Describe("Fungible with Auditor ne Issuer", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, t.ReplicationFactor, "alice", "bob")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", Label("T1"), func() { fungible.TestMixed(context.Background(), ts.II, nil, selector) })
		})
	}
},
)

func newTestSuite(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(StartPortDlog, Topology(common.Opts{
		CommType:        commType,
		SDKs:            []nodepkg.SDK{&fall.SDK{}},
		ReplicationOpts: opts,
	}))
	return ts, selector
}
