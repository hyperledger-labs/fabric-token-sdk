/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	nodepkg "github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/fabtokenv1"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/ffabtoken"
	dvp2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	for _, t := range integration.AllTestTypes {
		Describe("Plain DVP", t.Label, func() {
			ts, selector := newTestSuite(t.CommType, t.ReplicationFactor, "buyer", "seller")
			BeforeEach(ts.Setup)
			AfterEach(ts.TearDown)
			It("succeeded", func() { dvp2.TestAll(ts.II, selector) })
		})
	}
})

func newTestSuite(commType fsc.P2PCommunicationType, factor int, names ...string) (*token2.TestSuite, *token2.ReplicaSelector) {
	opts, selector := token2.NewReplicationOptions(factor, names...)
	ts := token2.NewTestSuite(StartPort, dvp2.Topology(dvp2.Opts{
		CommType:       commType,
		DefaultTMSOpts: common.TMSOpts{TokenSDKDriver: fabtokenv1.DriverIdentifier},
		// FSCLogSpec:     "token-sdk=debug:fabric-sdk=debug:info",
		SDKs:        []nodepkg.SDK{&ffabtoken.SDK{}},
		Replication: opts,
	}))
	return ts, selector
}
