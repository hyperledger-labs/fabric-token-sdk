/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mixed

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			network, err = integration.New(StartPortDlog(), "", Topology(&fabric3.SDK{}, &sdk.SDK{})...)
			Expect(err).NotTo(HaveOccurred())
			network.DeleteOnStop = false
			network.DeleteOnStart = true
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestMixed(network)
		})

	})

})
