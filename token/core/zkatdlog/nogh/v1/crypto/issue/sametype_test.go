/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue_test

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/issue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Issued Token Correctness", func() {
	var (
		verifier *issue.SameTypeVerifier
		prover   *issue.SameTypeProver
	)
	BeforeEach(func() {
		prover, verifier = GetSameTypeProverAndVerifier()
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
			})
		})
	})
	Describe("Verify", func() {
		It("Succeeds", func() {
			proof, err := prover.Prove()
			Expect(err).NotTo(HaveOccurred())
			err = verifier.Verify(proof)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func prepareTokens(pp []*math.G1) []*math.G1 {
	curve := math.Curves[1]
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())

	bf := make([]*math.Zr, 2)
	values := make([]uint64, 2)

	for i := 0; i < 2; i++ {
		bf[i] = curve.NewRandomZr(rand)
	}
	values[0] = 100
	values[1] = 50

	tokens := make([]*math.G1, len(values))
	for i := 0; i < len(values); i++ {
		tokens[i] = NewToken(curve.NewZrFromInt(int64(values[i])), bf[i], "ABC", pp, curve)
	}
	return tokens
}

func GetSameTypeProverAndVerifier() (*issue.SameTypeProver, *issue.SameTypeVerifier) {
	pp := preparePedersenParameters()
	curve := math.Curves[1]

	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())
	blindingFactor := curve.NewRandomZr(rand)
	com := pp[0].Mul(curve.HashToZr([]byte("ABC")))
	com.Add(pp[2].Mul(blindingFactor))

	tokens := prepareTokens(pp)
	return issue.NewSameTypeProver("ABC", blindingFactor, com, pp, math.Curves[1]), issue.NewSameTypeVerifier(tokens, pp, math.Curves[1])
}

func preparePedersenParameters() []*math.G1 {
	curve := math.Curves[1]
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*math.G1, 3)

	for i := 0; i < 3; i++ {
		pp[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return pp
}

func NewToken(value *math.Zr, rand *math.Zr, ttype string, pp []*math.G1, curve *math.Curve) *math.G1 {
	token := curve.NewG1()
	token.Add(pp[0].Mul(curve.HashToZr([]byte(ttype))))
	token.Add(pp[1].Mul(value))
	token.Add(pp[2].Mul(rand))
	return token
}
