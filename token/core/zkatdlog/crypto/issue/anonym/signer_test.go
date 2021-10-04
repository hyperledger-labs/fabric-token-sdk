/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package anonym_test

import (
	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/anonym"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Issuer NYM", func() {

	var (
		verifier *anonym.Verifier
		signer   *anonym.Signer
		pp       []*math.G1
	)
	BeforeEach(func() {
		pp = getPedersenParameters(3)
	})

	Describe("Signer", func() {
		When("signature is generated correctly", func() {
			BeforeEach(func() {
				signer = getIssuerSigner(2, 8, 3, pp)
				verifier = signer.Verifier
			})
			It("succeeds", func() {
				sig, err := signer.Sign([]byte("message"))
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())
				Expect(err).NotTo(HaveOccurred())
				err = verifier.Verify([]byte("message"), sig)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func GetIssuers(N, index int, pk *math.G1, pp []*math.G1, c *math.Curve) []*math.G1 {
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())
	issuers := make([]*math.G1, N)
	issuers[index] = pk
	for i := 0; i < N; i++ {
		if i != index {
			sk := c.NewRandomZr(rand)
			t := c.NewRandomZr(rand)
			issuers[i] = pp[0].Mul(sk)
			issuers[i].Add(pp[1].Mul(t))
		}
	}

	return issuers

}

func getIssuerSigner(index, N, bitlength int, pp []*math.G1) *anonym.Signer {
	c := math.Curves[1]
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())
	r := make([]*math.Zr, 3)
	bf := make([]*math.Zr, 2)
	for i := 0; i < len(r); i++ {
		r[i] = c.NewRandomZr(rand)
	}

	for i := 0; i < len(bf); i++ {
		bf[i] = c.NewRandomZr(rand)
	}
	pk := pp[0].Mul(r[0])
	pk.Add(pp[1].Mul(r[1]))

	issuers := GetIssuers(N, index, pk, pp, c)

	issuer := &anonym.Authorization{}
	issuer.Type = issuers[index].Copy()
	issuer.Type.Add(pp[2].Mul(bf[0]))

	issuer.Token = pp[0].Mul(r[1])
	issuer.Token.Add(pp[1].Mul(r[2]))
	issuer.Token.Add(pp[2].Mul(bf[1]))

	witness := anonym.NewWitness(r[0], r[1], r[2], bf[0], bf[1], index)

	return anonym.NewSigner(witness, issuers, issuer, bitlength, pp, c)

}
