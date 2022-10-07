/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"io/ioutil"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
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
			network, err = integration.New(StartPortDlog(), "", fungible.Topology("fabric", "dlog", false)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "auditor")
		})

		It("Update public params", func() {
			auditorId := fungible.GetAuditorIdentity(network, "newAuditor")
			issuerId := fungible.GetIssuerIdentity(network, "newIssuer.id1")
			p := network.Ctx.PlatformsByName["token"]

			tms := fungible.GetTms(network, "default")
			Expect(tms).NotTo(BeNil())

			nh := p.(*token.Platform).NetworkHandlers[p.(*token.Platform).Context.TopologyByName(tms.Network).Type()]
			ppBytes, err := ioutil.ReadFile(nh.(*fabric.NetworkHandler).TokenPlatform.PublicParametersFile(tms))
			Expect(err).NotTo(HaveOccurred())

			pp, err := crypto.NewPublicParamsFromBytes(ppBytes, crypto.DLogPublicParameters)
			Expect(err).NotTo(HaveOccurred())

			pp.Auditor = auditorId
			pp.Issuers = [][]byte{issuerId}

			ppBytes, err = pp.Serialize()
			Expect(err).NotTo(HaveOccurred())

			fungible.TestPublicParamsUpdate(network, "newAuditor", ppBytes, tms)
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
