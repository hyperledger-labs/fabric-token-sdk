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
		prover   *sigproof.MembershipProver
		verifier *sigproof.MembershipVerifier
	)
	Context("when the proof is computed correctly", func() {
		BeforeEach(func() {
			prover = getMembershipProver()
			verifier = prover.MembershipVerifier
		})
		It("Succeeds ", func() {
			proof, err := prover.Prove()
			Expect(err).NotTo(HaveOccurred())
			Expect(proof).NotTo(BeNil())
			err = verifier.Verify(proof)
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Context("when value does not correspond to signature", func() {
		BeforeEach(func() {
			prover = getBogusProver()
			verifier = prover.MembershipVerifier
		})
		It("fails", func() {
			proof, err := prover.Prove()
			Expect(err).NotTo(HaveOccurred())
			Expect(proof).NotTo(BeNil())
			err = verifier.Verify(proof)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid membership proof"))
		})
	})
})

func getMembershipProver() *sigproof.MembershipProver {
	signer := getSigner(1)
	sig, err := signer.Sign([]*bn256.Zr{bn256.NewZrInt(120)})
	Expect(err).NotTo(HaveOccurred())

	pp := preparePedersenParameters(2)
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	r := bn256.RandModOrder(rand)
	com, err := common.ComputePedersenCommitment([]*bn256.Zr{bn256.NewZrInt(120), r}, pp)
	Expect(err).NotTo(HaveOccurred())

	witness := sigproof.NewMembershipWitness(sig, bn256.NewZrInt(120), r)
	P := bn256.NewG1()
	return sigproof.NewMembershipProver(witness, com, P, signer.Q, signer.PK, pp)
}

func getBogusProver() *sigproof.MembershipProver {
	signer := getSigner(1)
	sig, err := signer.Sign([]*bn256.Zr{bn256.NewZrInt(120)})
	Expect(err).NotTo(HaveOccurred())

	pp := preparePedersenParameters(2)
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	r := bn256.RandModOrder(rand)
	com, err := common.ComputePedersenCommitment([]*bn256.Zr{bn256.NewZrInt(130), r}, pp)
	Expect(err).NotTo(HaveOccurred())

	err = signer.SignVerifier.Verify([]*bn256.Zr{bn256.NewZrInt(130), bn256.HashModOrder(bn256.NewZrInt(120).Bytes())}, sig)
	Expect(err).To(HaveOccurred())
	witness := sigproof.NewMembershipWitness(sig, bn256.NewZrInt(130), r)
	P := bn256.NewG1()
	return sigproof.NewMembershipProver(witness, com, P, signer.Q, signer.PK, pp)
}

func preparePedersenParameters(l int) []*bn256.G1 {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*bn256.G1, l)

	for i := 0; i < l; i++ {
		pp[i] = bn256.G1Gen().Mul(bn256.RandModOrder(rand))
	}
	return pp
}
