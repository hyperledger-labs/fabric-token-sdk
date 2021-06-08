/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Input/Output well formedness", func() {
	var (
		iow      *transfer.WellFormednessWitness
		pp       []*bn256.G1
		verifier *transfer.WellFormednessVerifier
		prover   *transfer.WellFormednessProver
		in       []*bn256.G1
		out      []*bn256.G1
		inBF     []*bn256.Zr
		outBF    []*bn256.Zr
	)
	BeforeEach(func() {
		pp = preparePedersenParameters()
		iow, in, out, inBF, outBF = prepareIOCProver(pp)
		prover = transfer.NewWellFormednessProver(iow, pp, in, out)
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				raw, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(raw).NotTo(BeNil())
				proof := &transfer.WellFormedness{}
				err = proof.Deserialize(raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(proof.Challenge).NotTo(BeNil())
				Expect(proof.Sum).NotTo(BeNil())
				Expect(proof.Type).NotTo(BeNil())
				Expect(len(proof.InputBlindingFactors)).To(Equal(2))
				Expect(len(proof.OutputBlindingFactors)).To(Equal(3))
				Expect(len(proof.OutputValues)).To(Equal(3))
				Expect(len(proof.InputValues)).To(Equal(2))
			})
		})
	})
	Describe("Verify", func() {
		BeforeEach(func() {
			verifier = transfer.NewWellFormednessVerifier(pp, in, out)
		})
		Context("The proof is generated honestly", func() {
			It("Succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verify
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("The proof is not generated correctly: wrong type", func() {
			It("fails", func() {
				// change type encoded in the commitments
				token := prepareToken(bn256.NewZrInt(100), inBF[0], "XYZ", pp)
				verifier.Inputs[0] = token
				// prover assumed to guess the type (e.g. ABC)
				prover.WellFormednessVerifier = verifier
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verification fails
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid zero-knowledge transfer"))
			})
		})
		Context("The proof is not generated correctly: wrong Values", func() {
			It("fails", func() {
				// change the value encoded in the input commitment
				token := prepareToken(bn256.NewZrInt(80), inBF[0], "ABC", pp)
				verifier.Inputs[0] = token

				// prover guess the value of the committed Values (e.g. 100)
				prover.WellFormednessVerifier = verifier
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verification fails
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid zero-knowledge transfer"))
			})
		})
		Context("The proof is not generated correctly: input sum != output sums", func() {
			It("fails", func() {
				// prover wants to increase the value of the output out of the blue
				token := prepareToken(bn256.NewZrInt(90), outBF[0], "ABC", pp)
				verifier.Outputs[0] = token
				// prover generates a proof
				prover.WellFormednessVerifier = verifier
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verification should fail
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid zero-knowledge transfer"))
			})
		})
		Context("The proof is not generated correctly: wrong blindingFactors", func() {
			It("fails", func() {
				// prover guess the blindingFactors
				rand, err := bn256.GetRand()
				Expect(err).NotTo(HaveOccurred())
				token := prepareToken(bn256.NewZrInt(100), bn256.RandModOrder(rand), "ABC", pp)
				verifier.Inputs[0] = token
				// prover generate proof
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verification fails
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid zero-knowledge transfer"))
			})
		})
	})
})

func preparePedersenParameters() []*bn256.G1 {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*bn256.G1, 3)

	for i := 0; i < 3; i++ {
		pp[i] = bn256.G1Gen().Mul(bn256.RandModOrder(rand))
	}
	return pp
}

func prepareIOCProver(pp []*bn256.G1) (*transfer.WellFormednessWitness, []*bn256.G1, []*bn256.G1, []*bn256.Zr, []*bn256.Zr) {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*bn256.Zr, 2)
	outBF := make([]*bn256.Zr, 3)
	inValues := make([]*bn256.Zr, 2)
	outValues := make([]*bn256.Zr, 3)
	for i := 0; i < 2; i++ {
		inBF[i] = bn256.RandModOrder(rand)
	}
	for i := 0; i < 3; i++ {
		outBF[i] = bn256.RandModOrder(rand)
	}
	ttype := "ABC"
	inValues[0] = bn256.NewZrInt(100)
	inValues[1] = bn256.NewZrInt(50)
	outValues[0] = bn256.NewZrInt(75)
	outValues[1] = bn256.NewZrInt(35)
	outValues[2] = bn256.NewZrInt(40)

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp)

	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}
	return transfer.NewWellFormednessWitness(intw, outtw), in, out, inBF, outBF
}

func prepareInputsOutputs(inValues, outValues, inBF, outBF []*bn256.Zr, ttype string, pp []*bn256.G1) ([]*bn256.G1, []*bn256.G1) {
	inputs := make([]*bn256.G1, len(inValues))
	outputs := make([]*bn256.G1, len(outValues))

	for i := 0; i < len(inputs); i++ {
		inputs[i] = bn256.NewG1()
		inputs[i].Add(pp[0].Mul(bn256.HashModOrder([]byte(ttype))))
		inputs[i].Add(pp[1].Mul(inValues[i]))
		inputs[i].Add(pp[2].Mul(inBF[i]))

	}

	for i := 0; i < len(outputs); i++ {
		outputs[i] = bn256.NewG1()
		outputs[i].Add(pp[0].Mul(bn256.HashModOrder([]byte(ttype))))
		outputs[i].Add(pp[1].Mul(outValues[i]))
		outputs[i].Add(pp[2].Mul(outBF[i]))

	}
	return inputs, outputs
}

func prepareToken(value *bn256.Zr, rand *bn256.Zr, ttype string, pp []*bn256.G1) *bn256.G1 {
	token := bn256.NewG1()
	token.Add(pp[0].Mul(bn256.HashModOrder([]byte(ttype))))
	token.Add(pp[1].Mul(value))
	token.Add(pp[2].Mul(rand))
	return token
}
