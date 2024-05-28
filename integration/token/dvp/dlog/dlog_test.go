/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	dvp2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	Describe("ZKAT-DLog DVP", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPort, dvp2.Topology(dvp2.Opts{
			CommType:       fsc.LibP2P,
			TokenSDKDriver: "dlog",
			FSCLogSpec:     "",
			SDKs:           []api2.SDK{&fabric3.SDK{}, &fdlog.SDK{}},
			Replication:    opts,
		}))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			dvp2.TestAll(ts.II, selector)
		})
	})
})
