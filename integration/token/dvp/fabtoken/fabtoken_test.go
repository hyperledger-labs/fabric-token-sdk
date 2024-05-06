/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	dvp2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	Describe("Plain DVP", func() {
		var ts = newTestSuite(fsc.LibP2P, integration.NoReplication)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			dvp2.TestAll(ts.II)
		})
	})
})

func newTestSuite(commType fsc.P2PCommunicationType, opts *integration.ReplicationOptions) *token2.TestSuite {
	return token2.NewTestSuite(opts.SQLConfigs, StartPort, dvp2.Topology(dvp2.Opts{
		CommType:       commType,
		TokenSDKDriver: "fabtoken",
		FSCLogSpec:     "",
		SDKs:           []api2.SDK{&fabric3.SDK{}, &sdk.SDK{}},
		Replication:    opts,
	}))
}
