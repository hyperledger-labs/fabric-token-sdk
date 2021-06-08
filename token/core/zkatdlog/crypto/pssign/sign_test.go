/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pointcheval Sanders signatures", func() {
	var (
		verifier *pssign.SignVerifier
		signer   *pssign.Signer
		m        []*bn256.Zr
	)
	Describe("Sign", func() {
		Context("KeyGen is initialized correctly", func() {
			It("Succeeds", func() {
				signer = &pssign.Signer{}
				err := signer.KeyGen(3)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(signer.SK)).To(Equal(5))
				Expect(len(signer.PK)).To(Equal(5))
				m = getMessages(3)
				sig, err := signer.Sign(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())
			})
		})
	})
	Describe("Verify", func() {
		Context("When the signature on message m is valid", func() {
			It("Succeeds", func() {
				sig, err := signer.Sign(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())
				bytes, err := signer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = signer.Deserialize(bytes)
				Expect(err).NotTo(HaveOccurred())
				verifier = signer.SignVerifier
				err = verifier.Verify(append(m, hashMessages(m)), sig)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("When the signature is randomized", func() {
			It("Succeeds", func() {
				sig, err := signer.Sign(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())

				err = sig.Randomize()
				Expect(err).NotTo(HaveOccurred())
				verifier = signer.SignVerifier
				err = verifier.Verify(append(m, hashMessages(m)), sig)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("When the signature is not on message m", func() {
			It("fails", func() {
				sig, err := signer.Sign(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())
				msg := getMessages(1)
				m[1] = msg[0]
				verifier = signer.SignVerifier
				err = verifier.Verify(append(m, hashMessages(m)), sig)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid Pointcheval-Sanders signature"))
			})
		})

	})
})

func getMessages(length int) []*bn256.Zr {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	m := make([]*bn256.Zr, length)
	for i := 0; i < length; i++ {
		m[i] = bn256.RandModOrder(rand)
	}
	return m
}

func hashMessages(m []*bn256.Zr) *bn256.Zr {
	var bytesToHash []byte
	for i := 0; i < len(m); i++ {
		bytes := m[i].Bytes()
		bytesToHash = append(bytesToHash, bytes...)
	}

	return bn256.HashModOrder(bytesToHash)
}
