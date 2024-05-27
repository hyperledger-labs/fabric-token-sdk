/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	orion "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/odlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/nft"
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

	Describe("NFT Orion", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", nft.Topology(common.Opts{
				Backend:        "orion",
				TokenSDKDriver: "dlog",
				SDKs:           []api.SDK{&orion.SDK{}, &odlog.SDK{}},
			})...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			nft.TestAll(network)
		})
	})

})
