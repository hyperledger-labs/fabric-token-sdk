/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	integration2 "github.com/hyperledger-labs/fabric-token-sdk/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/ffabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/multisig"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration2.AllTestTypes {
		Describe("Multisig", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, t.ReplicationFactor, "alice", "bob", "charlie", "dave")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", func() { multisig.TestAll(ts.II, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(StartPortDlog, multisig.Topology(
		common.Opts{
			Backend:         "fabric",
			CommType:        commType,
			DefaultTMSOpts:  common.TMSOpts{TokenSDKDriver: "fabtoken", Aries: false},
			SDKs:            []api.SDK{&ffabtoken.SDK{}},
			ReplicationOpts: opts,
			WebEnabled:      true, // Needed for the Remote Wallet with websockets
			FSCLogSpec:      "token-sdk=debug:fabric-sdk=debug:info",
			OnlyUnity:       true,
		},
	))
	return ts, selector
}
