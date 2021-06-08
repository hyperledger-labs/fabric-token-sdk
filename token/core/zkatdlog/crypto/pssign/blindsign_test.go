/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package pssign_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pointcheval Sanders blind signatures", func() {
	var (
		recipient *pssign.Recipient
		signer    *pssign.BlindSigner
		messages  []*bn256.Zr
		pp        []*bn256.G1
	)
	Describe("Prove", func() {
		BeforeEach(func() {
			rand, err := bn256.GetRand()
			Expect(err).NotTo(HaveOccurred())

			x := bn256.RandModOrder(rand)
			messages = getMessages(4)
			bf := bn256.RandModOrder(rand)
			pp = getPedersenParameters(5)

			com, err := common.ComputePedersenCommitment(append(messages, bf), pp)
			Expect(err).NotTo(HaveOccurred())

			recipient = pssign.NewRecipient(messages, bf, com, x, bn256.G1Gen(), bn256.G1Gen().Mul(x), pp, nil, nil)
			signer = &pssign.BlindSigner{Signer: getSigner(4), PedersenParameters: pp}

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

func getPedersenParameters(length int) []*bn256.G1 {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	var pp []*bn256.G1
	for i := 0; i < length; i++ {
		pp = append(pp, bn256.G1Gen().Mul(bn256.RandModOrder(rand)))
	}
	return pp
}

func getSigner(length int) *pssign.Signer {
	s := &pssign.Signer{}
	s.KeyGen(length)
	return s
}
