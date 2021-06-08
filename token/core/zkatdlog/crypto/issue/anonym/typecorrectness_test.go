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

var _ = Describe("Issuer TypeCorrectness", func() {

	var (
		prover   *anonym.TypeCorrectnessProver
		pp       []*bn256.G1
		verifier *anonym.TypeCorrectnessVerifier
	)
	BeforeEach(func() {
		pp = getPedersenParameters(3)

	})

	Describe("Prover", func() {
		When("proof is generated correctly", func() {
			BeforeEach(func() {
				prover = newTypeCorrectnessProver(pp)
				verifier = prover.TypeCorrectnessVerifier

			})
			It("succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func newTypeCorrectnessProver(pp []*bn256.G1) *anonym.TypeCorrectnessProver {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	bf := make([]*bn256.Zr, 2)
	for i := 0; i < len(bf); i++ {
		bf[i] = bn256.RandModOrder(rand)
	}

	opening := make([]*bn256.Zr, 3)
	for i := 0; i < len(opening); i++ {
		opening[i] = bn256.RandModOrder(rand)
	}

	tnym := pp[0].Mul(opening[0])   // SK
	tnym.Add(pp[1].Mul(opening[1])) // type
	tnym.Add(pp[2].Mul(bf[0]))

	token := pp[0].Mul(opening[1])   //type
	token.Add(pp[1].Mul(opening[2])) // Value
	token.Add(pp[2].Mul(bf[1]))

	witness := anonym.NewTypeCorrectnessWitness(opening[0], opening[1], opening[2], bf[0], bf[1])

	return anonym.NewTypeCorrectnessProver(witness, tnym, token, []byte("message"), pp)
}

func getPedersenParameters(l int) []*bn256.G1 {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	pp := make([]*bn256.G1, l)
	for i := 0; i < l; i++ {
		pp[i] = bn256.G1Gen().Mul(bn256.RandModOrder(rand))
	}
	return pp
}
