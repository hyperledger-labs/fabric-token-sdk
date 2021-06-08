/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
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
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	signer := getSigner(4)
	var messages []*bn256.Zr
	messages = append(messages, bn256.RandModOrder(rand), bn256.RandModOrder(rand), bn256.RandModOrder(rand), bn256.RandModOrder(rand))
	sig, err := signer.Sign(messages)
	Expect(err).NotTo(HaveOccurred())
	hash := sigproof.HashMessages(messages)
	err = signer.SignVerifier.Verify(append(messages, hash), sig)
	Expect(err).NotTo(HaveOccurred())

	pp := preparePedersenParameters(4)
	r := bn256.RandModOrder(rand)
	com, err := common.ComputePedersenCommitment([]*bn256.Zr{messages[0], messages[1], messages[2], r}, pp)
	Expect(err).NotTo(HaveOccurred())
	hidden := []*bn256.Zr{
		messages[0],
		messages[1],
		messages[2],
	}
	P := bn256.NewG1()

	return sigproof.NewSigProver(hidden, []*bn256.Zr{messages[3]}, sig, hash, r, com, []int{0, 1, 2}, []int{3}, P, signer.Q, signer.PK, pp)
}
