/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package rangeproof_test

import (
	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	rp "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/range"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("range proof", func() {
	var (
		prover   *rp.Prover
		verifier *rp.Verifier
	)
	BeforeEach(func() {
		prover = getRangeProver()
		verifier = prover.Verifier
	})
	Context("when prove parameters are correct", func() {
		It("Succeeds ", func() {
			proof, err := prover.Prove()
			Expect(err).NotTo(HaveOccurred())
			Expect(proof).NotTo(BeNil())
			err = verifier.Verify(proof)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func getRangeProver() *rp.Prover {
	c := math.Curves[1]
	signatures := make([]*pssign.Signature, 2)
	signer := getSigner(1)
	signatures[0], _ = signer.Sign([]*math.Zr{c.NewZrFromInt(0)})
	signatures[1], _ = signer.Sign([]*math.Zr{c.NewZrFromInt(1)})

	pp := preparePedersenParameters()
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	value := c.NewZrFromInt(3)
	bf := c.NewRandomZr(rand)

	tok := c.NewG1()
	tok.Add(pp[0].Mul(c.HashToZr([]byte("ABC"))))
	tok.Add(pp[1].Mul(value))
	tok.Add(pp[2].Mul(bf))

	tw := &token.TokenDataWitness{Value: value, Type: "ABC", BlindingFactor: bf}

	prover := rp.NewProver([]*token.TokenDataWitness{tw}, []*math.G1{tok}, signatures, 2, pp, signer.PK, c.GenG1, signer.Q, c)

	return prover
}

func getSigner(length int) *pssign.Signer {
	s := pssign.NewSigner(nil, nil, nil, math.Curves[1])
	s.KeyGen(length)
	return s
}

func preparePedersenParameters() []*math.G1 {
	curve := math.Curves[1]
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*math.G1, 3)

	for i := 0; i < 3; i++ {
		pp[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	return pp
}
