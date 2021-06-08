/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package anonym_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/anonym"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Issuer NYM", func() {

	var (
		verifier *anonym.Verifier
		signer   *anonym.Signer
		pp       []*bn256.G1
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

func GetIssuers(N, index int, pk *bn256.G1, pp []*bn256.G1) []*bn256.G1 {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	issuers := make([]*bn256.G1, N)
	issuers[index] = pk
	for i := 0; i < N; i++ {
		if i != index {
			sk := bn256.RandModOrder(rand)
			t := bn256.RandModOrder(rand)
			issuers[i] = pp[0].Mul(sk)
			issuers[i].Add(pp[1].Mul(t))
		}
	}

	return issuers

}

func getIssuerSigner(index, N, bitlength int, pp []*bn256.G1) *anonym.Signer {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	r := make([]*bn256.Zr, 3)
	bf := make([]*bn256.Zr, 2)
	for i := 0; i < len(r); i++ {
		r[i] = bn256.RandModOrder(rand)
	}

	for i := 0; i < len(bf); i++ {
		bf[i] = bn256.RandModOrder(rand)
	}
	pk := pp[0].Mul(r[0])
	pk.Add(pp[1].Mul(r[1]))

	issuers := GetIssuers(N, index, pk, pp)

	issuer := &anonym.Authorization{}
	issuer.Type = bn256.NewG1()
	issuer.Type.Copy(issuers[index])
	issuer.Type.Add(pp[2].Mul(bf[0]))

	issuer.Token = pp[0].Mul(r[1])
	issuer.Token.Add(pp[1].Mul(r[2]))
	issuer.Token.Add(pp[2].Mul(bf[1]))

	witness := anonym.NewWitness(r[0], r[1], r[2], bf[0], bf[1], index)

	return anonym.NewSigner(witness, issuers, issuer, bitlength, pp)

}
