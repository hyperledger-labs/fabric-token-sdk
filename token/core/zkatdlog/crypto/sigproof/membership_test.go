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
	c := math.Curves[1]
	signer := getSigner(1, c)
	sig, err := signer.Sign([]*math.Zr{c.NewZrFromInt(120)})
	Expect(err).NotTo(HaveOccurred())

	pp := preparePedersenParameters(2, c)
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	r := c.NewRandomZr(rand)
	com, err := common.ComputePedersenCommitment([]*math.Zr{c.NewZrFromInt(120), r}, pp, c)
	Expect(err).NotTo(HaveOccurred())

	witness := sigproof.NewMembershipWitness(sig, c.NewZrFromInt(120), r)
	P := c.NewG1()
	return sigproof.NewMembershipProver(witness, com, P, signer.Q, signer.PK, pp, c)
}

func getBogusProver() *sigproof.MembershipProver {
	c := math.Curves[1]
	signer := getSigner(1, c)
	sig, err := signer.Sign([]*math.Zr{c.NewZrFromInt(120)})
	Expect(err).NotTo(HaveOccurred())

	pp := preparePedersenParameters(2, c)
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	r := c.NewRandomZr(rand)
	com, err := common.ComputePedersenCommitment([]*math.Zr{c.NewZrFromInt(130), r}, pp, c)
	Expect(err).NotTo(HaveOccurred())

	err = signer.SignVerifier.Verify([]*math.Zr{c.NewZrFromInt(130), c.HashToZr(c.NewZrFromInt(120).Bytes())}, sig)
	Expect(err).To(HaveOccurred())
	witness := sigproof.NewMembershipWitness(sig, c.NewZrFromInt(130), r)
	P := c.NewG1()
	return sigproof.NewMembershipProver(witness, com, P, signer.Q, signer.PK, pp, c)
}

func preparePedersenParameters(l int, curve *math.Curve) []*math.G1 {
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*math.G1, l)

	for i := 0; i < l; i++ {
		pp[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return pp
}
