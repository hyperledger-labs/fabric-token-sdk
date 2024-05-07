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
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Orion EndToEnd", func() {
	Describe("Orion ZKAT-DLog", func() {
		opts, selector := token2.NoReplication()
		var ts = newTestSuite(fsc.LibP2P, opts)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", nil, true, selector)
		})
	})

})

func newTestSuite(commType fsc.P2PCommunicationType, opts *token2.ReplicationOptions) *token2.TestSuite {
	return token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, topology.Topology(
		topology.Opts{
			Backend:        "orion",
			CommType:       commType,
			TokenSDKDriver: "dlog",
			Aries:          true,
			SDKs:           []api.SDK{&orion3.SDK{}, &sdk.SDK{}},
			Replication:    opts,
		},
	))
}
