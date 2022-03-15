/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	dvp2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EndToEnd", func() {
	var (
		ii *integration.Infrastructure
	)

	AfterEach(func() {
		ii.Stop()
	})

	Describe("Plain DVP", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(StartPort(), "", dvp2.Topology("fabtoken")...)
			Expect(err).NotTo(HaveOccurred())
			ii.RegisterPlatformFactory(token.NewPlatformFactory())
			ii.Generate()
			ii.Start()
		})

		It("succeeded", func() {
			dvp2.TestAll(ii)
		})
	})
})
