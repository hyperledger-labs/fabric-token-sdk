/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp"
)

var _ = Describe("EndToEnd", func() {
	var (
		network *integration.Infrastructure
	)

	AfterEach(func() {
		network.Stop()
	})

	Describe("Plain DVP", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.Generate(StartPort(), dvp.Topology("fabtoken")...)
			Expect(err).NotTo(HaveOccurred())
			network.Start()
		})

		It("succeeded", func() {
			dvp.TestAll(network)
		})
	})
})
