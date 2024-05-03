/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dloghsm

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	Describe("Fungible with HSM", func() {
		var ts = newTestSuite(fsc.LibP2P, false, integration.NoReplication)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", nil, true)
		})
	})

	Describe("Fungible with Auditor = Issuer with HSM", func() {
		var ts = newTestSuite(fsc.LibP2P, true, integration.NoReplication)
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "issuer", nil, true)
		})
	})

})

func newTestSuite(commType fsc.P2PCommunicationType, auditorAsIssuer bool, opts *integration.ReplicationOptions) *token2.TestSuite {
	return token2.NewTestSuite(opts.SQLConfigs, StartPortDlog, topology.Topology(
		topology.Opts{
			Backend:         "fabric",
			CommType:        commType,
			TokenSDKDriver:  "dlog",
			Aries:           true,
			HSM:             true,
			AuditorAsIssuer: auditorAsIssuer,
			SDKs:            []api.SDK{&fabric.SDK{}, &sdk.SDK{}},
			Replication:     &token2.ReplicationOptions{ReplicationOptions: opts},
		},
	))
}
