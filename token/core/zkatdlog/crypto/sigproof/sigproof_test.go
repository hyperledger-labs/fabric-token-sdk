/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof_test

import (
	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/sigproof"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("membership", func() {
	var (
		prover   *sigproof.SigProver
		verifier *sigproof.SigVerifier
	)
	Context("when the proof is computed correctly", func() {
		BeforeEach(func() {
			prover = getSignatureProver()
			verifier = prover.SigVerifier
		})
		It("Succeeds ", func() {
			proof, err := prover.Prove()
			Expect(err).NotTo(HaveOccurred())
			Expect(proof).NotTo(BeNil())
			err = verifier.Verify(proof)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func getSignatureProver() *sigproof.SigProver {
	curve := math.Curves[1]
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())
	signer := getSigner(4, curve)
	var messages []*math.Zr
	messages = append(messages, curve.NewRandomZr(rand), curve.NewRandomZr(rand), curve.NewRandomZr(rand), curve.NewRandomZr(rand))
	sig, err := signer.Sign(messages)
	Expect(err).NotTo(HaveOccurred())
	hash := sigproof.HashMessages(messages, curve)
	err = signer.SignVerifier.Verify(append(messages, hash), sig)
	Expect(err).NotTo(HaveOccurred())

	pp := preparePedersenParameters(4, curve)
	r := curve.NewRandomZr(rand)
	com, err := common.ComputePedersenCommitment([]*math.Zr{messages[0], messages[1], messages[2], r}, pp, curve)
	Expect(err).NotTo(HaveOccurred())
	hidden := []*math.Zr{
		messages[0],
		messages[1],
		messages[2],
	}
	P := curve.NewG1()

	return sigproof.NewSigProver(hidden, []*math.Zr{messages[3]}, sig, hash, r, com, []int{0, 1, 2}, []int{3}, P, signer.Q, signer.PK, pp, curve)
}
