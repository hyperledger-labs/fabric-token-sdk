/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package o2omp_test

import (
	"io"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/o2omp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	bn256 "github.ibm.com/fabric-research/mathlib"
)

var _ = Describe("One out of many proof", func() {
	var (
		commitments []*bn256.G1
		index       int
		randomness  *bn256.Zr
		pp          []*bn256.G1

		prover   *o2omp.Prover
		verifier *o2omp.Verifier
		rand     io.Reader
		curve    *bn256.Curve
	)
	BeforeEach(func() {
		curve = bn256.Curves[1]
		pp = getPedersenParameters(2, curve)
		var err error
		rand, err = curve.Rand()
		Expect(err).NotTo(HaveOccurred())
		randomness = curve.NewRandomZr(rand)
		index = 1
		commitments = computePedersenCommitments(pp, index, 4, randomness, curve)
		verifier = o2omp.NewVerifier(commitments, []byte("message to be signed"), pp, 2, curve)
	})
	Describe("Prover", func() {
		When("proof is generated correctly", func() {
			BeforeEach(func() {
				prover = o2omp.NewProver(commitments, []byte("message to be signed"), pp, 2, index, randomness, curve)

			})
			It("succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("proof is invalid", func() {
			BeforeEach(func() {
				coms := []*bn256.G1{commitments[1], commitments[0], commitments[2], commitments[3]}
				prover = o2omp.NewProver(coms, []byte("message to be signed"), pp, 2, index, randomness, curve)
			})
			It("fails", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("verification of first equation of one out of many proof failed"))
			})
		})
		When("prover does not know the correct randomness", func() {
			BeforeEach(func() {
				prover = o2omp.NewProver(commitments, []byte("message to be signed"), pp, 2, index, curve.NewRandomZr(rand), curve)

			})
			It("fails", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("verification of third equation of one out of many proof failed"))
			})
		})
	})
})

func getPedersenParameters(l int, curve *bn256.Curve) []*bn256.G1 {
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())
	pp := make([]*bn256.G1, l)
	for i := 0; i < l; i++ {
		pp[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return pp
}

func computePedersenCommitments(pp []*bn256.G1, index, N int, randomness *bn256.Zr, curve *bn256.Curve) []*bn256.G1 {
	com := make([]*bn256.G1, N)
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())
	for i := 0; i < N; i++ {
		if i != index {
			com[i] = pp[0].Mul(curve.NewRandomZr(rand))
			com[i].Add(pp[1].Mul(curve.NewRandomZr(rand)))
		} else {
			com[i] = pp[1].Mul(randomness)
		}
	}
	return com
}
