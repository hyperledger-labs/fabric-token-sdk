/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"
)

var _ = Describe("Stress EndToEnd", func() {
	var (
		network *integration.Infrastructure
	)

	AfterEach(func() {
		network.DeleteOnStop = false
		network.Stop()
	})

	Describe("T1 Fungible with Auditor ne Issuer", func() {
		opts, selector := token2.NewReplicationOptions(token2.None)
		BeforeEach(func() {
			// notice that fabric-ca does not support yet aries
			var err error
			network, err = integration.New(StartPortDlog(), "", integration.ReplaceTemplate(topology2.Topology(
				common.Opts{
					Backend:         "fabric",
					TokenSDKDriver:  "dlog",
					Aries:           true,
					ReplicationOpts: opts,
					//FSCLogSpec:     "token-sdk=debug:fabric-sdk=debug:info",
					SDKs:       []api.SDK{&fdlog.SDK{}},
					Monitoring: true,
				},
			))...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("stress", func() {
			fungible.TestStress(network, "auditor", selector)
		})
	})

})
