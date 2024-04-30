/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	fabric3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	dvp2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EndToEnd", func() {
	var (
		ii *integration.Infrastructure
	)

	AfterEach(func() {
		ii.Stop()
	})

	Describe("ZKAT-DLog DVP", func() {
		BeforeEach(func() {
			var err error
			ii, err = integration.New(StartPort(), "", dvp2.Topology("dlog", &fabric3.SDK{}, &sdk.SDK{})...)
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
