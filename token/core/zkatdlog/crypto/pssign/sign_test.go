/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign_test

import (
	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pointcheval Sanders signatures", func() {
	var (
		verifier *pssign.SignVerifier
		signer   *pssign.Signer
		m        []*math.Zr
		curve    *math.Curve
	)
	Describe("Sign", func() {
		BeforeEach(func() {
			curve = math.Curves[1]
		})
		Context("KeyGen is initialized correctly", func() {
			It("Succeeds", func() {
				signer = pssign.NewSigner(nil, nil, nil, curve)
				err := signer.KeyGen(3)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(signer.SK)).To(Equal(5))
				Expect(len(signer.PK)).To(Equal(5))
				m = getMessages(3, curve)
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
				err = verifier.Verify(append(m, hashMessages(m, verifier.Curve)), sig)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("When the signature is randomized", func() {
			It("Succeeds", func() {
				sig, err := signer.Sign(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())

				err = signer.Randomize(sig)
				Expect(err).NotTo(HaveOccurred())
				verifier = signer.SignVerifier
				err = verifier.Verify(append(m, hashMessages(m, verifier.Curve)), sig)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("When the signature is not on message m", func() {
			It("fails", func() {
				sig, err := signer.Sign(m)
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())
				msg := getMessages(1, signer.Curve)
				m[1] = msg[0]
				verifier = signer.SignVerifier
				err = verifier.Verify(append(m, hashMessages(m, verifier.Curve)), sig)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid Pointcheval-Sanders signature"))
			})
		})

	})
})

func getMessages(length int, curve *math.Curve) []*math.Zr {
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())
	m := make([]*math.Zr, length)
	for i := 0; i < length; i++ {
		m[i] = curve.NewRandomZr(rand)
	}
	return m
}

func hashMessages(m []*math.Zr, curve *math.Curve) *math.Zr {
	var bytesToHash []byte
	for i := 0; i < len(m); i++ {
		bytes := m[i].Bytes()
		bytesToHash = append(bytesToHash, bytes...)
	}

	return curve.HashToZr(bytesToHash)
}
