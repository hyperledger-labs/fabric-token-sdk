/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue_test

import (
	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Issued Token Correctness", func() {
	var (
		verifier *issue.WellFormednessVerifier
		prover   *issue.WellFormednessProver
	)
	BeforeEach(func() {
		prover = GetITCPProver()
		verifier = prover.WellFormednessVerifier
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				raw, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(raw).NotTo(BeNil())
				proof := &issue.WellFormedness{}
				err = proof.Deserialize(raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(proof.Challenge).NotTo(BeNil())
				Expect(len(proof.BlindingFactors)).To(Equal(2))
				Expect(len(proof.Values)).To(Equal(2))
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

func PrepareTokenWitness(pp []*math.G1) ([]*token.TokenDataWitness, []*math.G1, []*math.Zr) {
	curve := math.Curves[1]
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())

	bF := make([]*math.Zr, 2)
	values := make([]*math.Zr, 2)
	for i := 0; i < 2; i++ {
		bF[i] = curve.NewRandomZr(rand)
	}
	ttype := "ABC"
	values[0] = curve.NewZrFromInt(100)
	values[1] = curve.NewZrFromInt(50)

	tokens := PrepareTokens(values, bF, ttype, pp)
	return issue.NewTokenDataWitness(ttype, values, bF), tokens, bF
}

func PrepareTokens(values, bf []*math.Zr, ttype string, pp []*math.G1) []*math.G1 {
	curve := math.Curves[1]
	tokens := make([]*math.G1, len(values))
	for i := 0; i < len(values); i++ {
		tokens[i] = NewToken(values[i], bf[i], ttype, pp, curve)
	}
	return tokens
}

func GetITCPProver() *issue.WellFormednessProver {
	pp := preparePedersenParameters()
	tw, tokens, _ := PrepareTokenWitness(pp)

	return issue.NewWellFormednessProver(tw, tokens, false, pp, math.Curves[1])
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
