/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package rp_test

import (
	"strconv"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/rp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Inner Product Argument", func() {

	Describe("IPA", func() {
		Context("If IPA is generated correctly", func() {
			It("Succeeds", func() {
				curve := math.Curves[0]
				nr := uint64(6)
				l := uint64(1 << nr)
				leftGens := make([]*math.G1, l)
				rightGens := make([]*math.G1, l)
				left := make([]*math.Zr, l)
				right := make([]*math.Zr, l)
				rand, err := curve.Rand()
				Expect(err).NotTo(HaveOccurred())
				com := curve.NewG1()
				Q := curve.GenG1
				for i := 0; i < len(left); i++ {
					leftGens[i] = curve.HashToG1([]byte(strconv.Itoa(i)))
					rightGens[i] = curve.HashToG1([]byte(strconv.Itoa(i + 1)))
					left[i] = curve.NewRandomZr(rand)
					right[i] = curve.NewRandomZr(rand)
					com.Add(leftGens[i].Mul(left[i]))
					com.Add(rightGens[i].Mul(right[i]))
				}

				prover := rp.NewIPAProver(innerProduct(left, right, curve), left, right, Q, leftGens, rightGens, com, nr, curve)
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				verifier := rp.NewIPAVerifier(innerProduct(left, right, curve), Q, leftGens, rightGens, com, nr, curve)
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func innerProduct(left []*math.Zr, right []*math.Zr, c *math.Curve) *math.Zr {
	ip := c.NewZrFromInt(0)
	for i, l := range left {
		ip = c.ModAdd(ip, c.ModMul(l, right[i], c.GroupOrder), c.GroupOrder)
	}
	return ip
}
