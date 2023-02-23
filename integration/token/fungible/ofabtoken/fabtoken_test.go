/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Orion EndToEnd", func() {
	var (
		network *integration.Infrastructure
	)

	AfterEach(func() {
		//network.DeleteOnStop = false
		network.Stop()
	})

	Describe("Orion FabToken", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", topology.Topology("orion", "fabtoken", false)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "auditor", nil)
		})
	})

})
