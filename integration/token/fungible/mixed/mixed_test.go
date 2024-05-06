/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mixed

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	api2 "github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	Describe("Fungible with Auditor ne Issuer", func() {
		var ts = newTestSuite(fsc.LibP2P, integration.NoReplication)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)
		It("succeeded", func() { fungible.TestMixed(ts.II) })
	})
})

func newTestSuite(commType fsc.P2PCommunicationType, opts *integration.ReplicationOptions) *token2.TestSuite {
	return token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, Topology(Opts{
		CommType:    commType,
		FSCLogSpec:  "",
		SDKs:        []api2.SDK{&fabric3.SDK{}, &sdk.SDK{}},
		Replication: opts,
	}))
}
