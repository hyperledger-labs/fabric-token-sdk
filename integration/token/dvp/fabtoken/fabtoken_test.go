/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/ffabtoken"
	dvp2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	Describe("Plain DVP", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPort, dvp2.Topology(dvp2.Opts{
			CommType:       fsc.LibP2P,
			TokenSDKDriver: "fabtoken",
			FSCLogSpec:     "",
			SDKs:           []api2.SDK{&fabric3.SDK{}, &ffabtoken.SDK{}},
			Replication:    opts,
		}))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			dvp2.TestAll(ts.II, selector)
		})
	})
})
