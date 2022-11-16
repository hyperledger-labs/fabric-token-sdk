/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
)

var _ = Describe("EndToEnd", func() {
	var (
		network *integration.Infrastructure
	)

	AfterEach(func() {
		network.Stop()
	})

	Describe("Fungible with Auditor ne Issuer", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", fungible.Topology("fabric", "dlog", false)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "auditor")
		})
	})

	Describe("Fungible with Auditor = Issuer", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", fungible.Topology("fabric", "dlog", true)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "issuer")
		})
	})

})
