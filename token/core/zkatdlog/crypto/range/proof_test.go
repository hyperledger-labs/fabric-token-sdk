/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package rangeproof_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
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
	signatures := make([]*pssign.Signature, 2)
	signer := getSigner(1)
	signatures[0], _ = signer.Sign([]*bn256.Zr{bn256.NewZrInt(0)})
	signatures[1], _ = signer.Sign([]*bn256.Zr{bn256.NewZrInt(1)})

	pp := preparePedersenParameters()
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	value := bn256.NewZrInt(3)
	bf := bn256.RandModOrder(rand)

	tok := bn256.NewG1()
	tok.Add(pp[0].Mul(bn256.HashModOrder([]byte("ABC"))))
	tok.Add(pp[1].Mul(value))
	tok.Add(pp[2].Mul(bf))

	tw := &token.TokenDataWitness{Value: value, Type: "ABC", BlindingFactor: bf}

	prover := rp.NewProver([]*token.TokenDataWitness{tw}, []*bn256.G1{tok}, signatures, 2, pp, signer.PK, bn256.G1Gen(), signer.Q)

	return prover
}

func getSigner(length int) *pssign.Signer {
	s := &pssign.Signer{}
	s.KeyGen(length)
	return s
}

func preparePedersenParameters() []*bn256.G1 {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*bn256.G1, 3)

	for i := 0; i < 3; i++ {
		pp[i] = bn256.G1Gen().Mul(bn256.RandModOrder(rand))
	}
	return pp
}
