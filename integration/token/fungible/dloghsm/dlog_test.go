/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dloghsm

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	fabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
)

var _ = Describe("EndToEnd", func() {
	var (
		network *integration.Infrastructure
	)

	AfterEach(func() {
		network.Stop()
	})

	Describe("Fungible with HSM", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", topology.Topology(
				common.Opts{
					Backend:         "fabric",
					TokenSDKDriver:  "dlog",
					Aries:           true,
					HSM:             true,
					AuditorAsIssuer: false,
					//FSCLogSpec:     "token-sdk=debug:fabric-sdk=debug:info",
					SDKs: []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				})...,
			)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "auditor", nil, true)
		})
	})

	Describe("Fungible with Auditor = Issuer with HSM", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", topology.Topology(
				common.Opts{
					Backend:         "fabric",
					TokenSDKDriver:  "dlog",
					Aries:           true,
					HSM:             true,
					AuditorAsIssuer: true,
					//FSCLogSpec:     "token-sdk=debug:fabric-sdk=debug:info",
					SDKs: []api.SDK{&fabric.SDK{}, &fdlog.SDK{}},
				})...,
			)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "issuer", nil, true)
		})
	})

})
