/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package elgamal_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/elgamal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Elgamal encryption", func() {

	Describe("Encrypt", func() {
		Context("Encryption performed correctly", func() {
			It("Succeeds", func() {
				rand, err := bn256.GetRand()
				Expect(err).NotTo(HaveOccurred())
				x := bn256.RandModOrder(rand)
				SK := elgamal.NewSecretKey(x, bn256.G1Gen(), bn256.G1Gen().Mul(x))
				m := bn256.RandModOrder(rand)
				C, _, err := SK.PublicKey.Encrypt(SK.Gen.Mul(m))
				Expect(err).NotTo(HaveOccurred())
				Expect(SK.Decrypt(C).Equals(SK.Gen.Mul(m))).To(Equal(true))
			})
		})
	})
})
