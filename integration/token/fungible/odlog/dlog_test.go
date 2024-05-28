/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	orion3 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/odlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Orion EndToEnd", func() {
	Describe("Orion ZKAT-DLog", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, topology.Topology(
			common.Opts{
				Backend:         "orion",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				Aries:           true,
				SDKs:            []api.SDK{&orion3.SDK{}, &odlog.SDK{}},
				ReplicationOpts: opts,
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", nil, true, selector)
		})
	})

})
