/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mixed

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fall"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	Describe("Fungible with Auditor ne Issuer", func() {
		opts, selector := token2.NoReplication()
		ts := token2.NewTestSuite(nil, StartPortDlog, Topology(common.Opts{
			CommType:        fsc.LibP2P,
			FSCLogSpec:      "",
			SDKs:            []api2.SDK{&fabric3.SDK{}, &fall.SDK{}},
			ReplicationOpts: opts,
		}))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)
		It("succeeded", func() { fungible.TestMixed(ts.II, selector) })
	})
})
