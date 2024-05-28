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
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("EndToEnd", func() {
	Describe("Fungible with HSM", func() {
		ts := token2.NewTestSuite(nil, StartPortDlog, topology.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				Aries:           true,
				HSM:             true,
				SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				ReplicationOpts: integration.NoReplication,
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "auditor", nil, true)
		})
	})

	Describe("Fungible with Auditor = Issuer with HSM", func() {
		ts := token2.NewTestSuite(nil, StartPortDlog, topology.Topology(
			common.Opts{
				Backend:         "fabric",
				CommType:        fsc.LibP2P,
				TokenSDKDriver:  "dlog",
				Aries:           true,
				HSM:             true,
				AuditorAsIssuer: true,
				SDKs:            []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				ReplicationOpts: integration.NoReplication,
			},
		))
		BeforeEach(ts.Setup)
		AfterEach(ts.TearDown)

		It("succeeded", func() {
			fungible.TestAll(ts.II, "issuer", nil, true)
		})
	})

})
