/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package elgamal_test

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/elgamal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Elgamal encryption", func() {

	Describe("Encrypt", func() {
		Context("Encryption performed correctly", func() {
			It("Succeeds", func() {
				curve := math.Curves[0]
				rand, err := curve.Rand()
				Expect(err).NotTo(HaveOccurred())
				x := curve.NewRandomZr(rand)
				SK := elgamal.NewSecretKey(x, curve.GenG1, curve.GenG1.Mul(x), curve)
				m := curve.NewRandomZr(rand)
				C, _, err := SK.PublicKey.Encrypt(SK.Gen.Mul(m))
				Expect(err).NotTo(HaveOccurred())
				M, err := SK.Decrypt(C)
				Expect(err).NotTo(HaveOccurred())
				Expect(M.Equals(SK.Gen.Mul(m))).To(Equal(true))
			})
		})
	})
})
