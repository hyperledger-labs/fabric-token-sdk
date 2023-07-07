/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/topology"
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
			network, err = integration.New(StartPortDlog(), "", topology2.Topology("fabric", "dlog", false)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "auditor", nil)
		})

		It("Update public params", func() {
			tms := fungible.GetTMS(network, "default")
			fungible.TestPublicParamsUpdate(network, "newAuditor", PrepareUpdatedPublicParams(network, "newAuditor", tms), tms, false)
		})

		It("Test Identity Revocation", func() {
			fungible.RegisterAuditor(network, "auditor", nil)
			rId := fungible.GetRevocationHandle(network, "bob")
			fungible.TestRevokeIdentity(network, "auditor", rId, hash.Hashable(rId).String()+" Identity is in revoked state")
		})

		It("Test Remote Wallet", func() {
			fungible.TestRemoteOwnerWallet(network, "auditor")
		})
	})

	Describe("Fungible with Auditor = Issuer", func() {
		BeforeEach(func() {
			var err error
			network, err = integration.New(StartPortDlog(), "", topology2.Topology("fabric", "dlog", true)...)
			Expect(err).NotTo(HaveOccurred())
			network.RegisterPlatformFactory(token.NewPlatformFactory())
			network.Generate()
			network.Start()
		})

		It("succeeded", func() {
			fungible.TestAll(network, "issuer", nil)
		})

		It("Update public params", func() {
			tms := fungible.GetTMS(network, "default")
			fungible.TestPublicParamsUpdate(network, "newIssuer", PrepareUpdatedPublicParams(network, "newIssuer", tms), tms, true)
		})

	})

})

func PrepareUpdatedPublicParams(network *integration.Infrastructure, auditor string, tms *topology.TMS) []byte {
	auditorId := fungible.GetAuditorIdentity(network, auditor)
	issuerId := fungible.GetIssuerIdentity(network, "newIssuer.id1")
	tokenPlatform, ok := network.Ctx.PlatformsByName["token"].(*token.Platform)
	Expect(ok).To(BeTrue(), "failed to get token platform from context")

	// Deserialize current params
	ppBytes, err := os.ReadFile(tokenPlatform.PublicParametersFile(tms))
	Expect(err).NotTo(HaveOccurred())
	pp, err := crypto.NewPublicParamsFromBytes(ppBytes, crypto.DLogPublicParameters)
	Expect(err).NotTo(HaveOccurred())
	Expect(pp.Validate()).NotTo(HaveOccurred())

	// Update PP
	pp.Auditor = auditorId
	pp.Issuers = [][]byte{issuerId}

	// Serialize
	ppBytes, err = pp.Serialize()
	Expect(err).NotTo(HaveOccurred())

	return ppBytes
}
