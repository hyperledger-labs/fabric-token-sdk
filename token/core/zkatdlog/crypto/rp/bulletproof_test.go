/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp_test

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/rp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	math2 "math"
	"strconv"
)

var _ = Describe("Range Proof", func() {

	Describe("IPA", func() {
		Context("If IPA is generated correctly", func() {
			It("Succeeds", func() {
				curve := math.Curves[1]
				nr := 3
				l := int(math2.Pow(2, float64(nr)))
				leftGens := make([]*math.G1, l)
				rightGens := make([]*math.G1, l)

				rand, err := curve.Rand()
				Expect(err).NotTo(HaveOccurred())

				Q := curve.GenG1.Mul(curve.NewRandomZr(rand))
				P := curve.GenG1.Mul(curve.NewRandomZr(rand))
				H := curve.GenG1.Mul(curve.NewRandomZr(rand))
				G := curve.GenG1.Mul(curve.NewRandomZr(rand))
				for i := 0; i < len(leftGens); i++ {
					leftGens[i] = curve.HashToG1([]byte(strconv.Itoa(i)))
					rightGens[i] = curve.HashToG1([]byte(strconv.Itoa(i + 1)))
				}
				bf := curve.NewRandomZr(rand)
				com := G.Mul(curve.NewZrFromInt(115))
				com.Add(H.Mul(bf))
				prover := rp.NewRangeProver(com, 115, []*math.G1{G, H}, bf, leftGens, rightGens, P, Q, nr, l, curve)
				verifier := rp.NewRangeVerifier(com, []*math.G1{G, H}, leftGens, rightGens, P, Q, nr, l, curve)

				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())

			})
		})
	})
})
