/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign_test

import (
	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pointcheval Sanders blind signatures", func() {
	var (
		recipient *pssign.Recipient
		signer    *pssign.BlindSigner
		messages  []*math.Zr
		pp        []*math.G1
		curve     *math.Curve
	)
	Describe("Prove", func() {
		BeforeEach(func() {
			curve = math.Curves[1]
			rand, err := curve.Rand()
			Expect(err).NotTo(HaveOccurred())

			x := curve.NewRandomZr(rand)
			messages = getMessages(4, curve)
			bf := curve.NewRandomZr(rand)
			pp = getPedersenParameters(5, curve)

			com, err := common.ComputePedersenCommitment(append(messages, bf), pp, curve)
			Expect(err).NotTo(HaveOccurred())

			recipient = pssign.NewRecipient(messages, bf, com, x, curve.GenG1, curve.GenG1.Mul(x), pp, nil, nil, curve)
			signer = &pssign.BlindSigner{Signer: getSigner(4, curve), PedersenParameters: pp}

		})
		Context("signature request is generated correctly", func() {
			It("succeeds", func() {
				req, err := recipient.GenerateBlindSignRequest()
				Expect(err).NotTo(HaveOccurred())
				Expect(req).NotTo(BeNil())
				res, err := signer.BlindSign(req)
				Expect(err).NotTo(HaveOccurred())
				recipient.SignVerifier = signer.SignVerifier
				sig, err := recipient.VerifyResponse(res)
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).NotTo(BeNil())
			})
		})
	})
})

func getPedersenParameters(length int, curve *math.Curve) []*math.G1 {
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())
	var pp []*math.G1
	for i := 0; i < length; i++ {
		pp = append(pp, curve.GenG1.Mul(curve.NewRandomZr(rand)))
	}
	return pp
}

func getSigner(length int, curve *math.Curve) *pssign.Signer {
	s := pssign.NewSigner(nil, nil, nil, curve)
	s.KeyGen(length)
	return s
}
