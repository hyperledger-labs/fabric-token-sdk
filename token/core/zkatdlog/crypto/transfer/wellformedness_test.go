/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"sync"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Input/Output well formedness", func() {
	var (
		iow         *transfer.WellFormednessWitness
		pp          []*math.G1
		verifier    *transfer.WellFormednessVerifier
		prover      *transfer.WellFormednessProver
		in          []*math.G1
		out         []*math.G1
		inBF        []*math.Zr
		outBF       []*math.Zr
		c           *math.Curve
		parallelism = 1000
	)
	BeforeEach(func() {
		c = math.Curves[1]
		pp = preparePedersenParameters(c)
		iow, in, out, inBF, outBF = prepareIOCProver(pp, c)
		prover = transfer.NewWellFormednessProver(iow, pp, in, out, c)
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
			verifier = transfer.NewWellFormednessVerifier(pp, in, out, c)
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
		Context("The proof is generated and verified in parallel", func() {
			It("Succeeds", func() {
				var wg sync.WaitGroup
				wg.Add(parallelism)

				for i := 0; i < parallelism; i++ {
					go func() {
						defer wg.Done()
						proof, err := prover.Prove()
						Expect(err).NotTo(HaveOccurred())

						// verify
						err = verifier.Verify(proof)
						Expect(err).NotTo(HaveOccurred())
					}()
				}

				wg.Wait()
			})
		})

		Context("The proof is not generated correctly: wrong type", func() {
			It("fails", func() {
				// change type encoded in the commitments
				token := prepareToken(c.NewZrFromInt(100), inBF[0], "XYZ", pp, c)
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
				token := prepareToken(c.NewZrFromInt(80), inBF[0], "ABC", pp, c)
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
				token := prepareToken(c.NewZrFromInt(90), outBF[0], "ABC", pp, c)
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
				rand, err := c.Rand()
				Expect(err).NotTo(HaveOccurred())
				token := prepareToken(c.NewZrFromInt(100), c.NewRandomZr(rand), "ABC", pp, c)
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

func preparePedersenParameters(c *math.Curve) []*math.G1 {
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*math.G1, 3)

	for i := 0; i < 3; i++ {
		pp[i] = c.GenG1.Mul(c.NewRandomZr(rand))
	}
	return pp
}

func prepareIOCProver(pp []*math.G1, c *math.Curve) (*transfer.WellFormednessWitness, []*math.G1, []*math.G1, []*math.Zr, []*math.Zr) {
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 3)
	inValues := make([]*math.Zr, 2)
	outValues := make([]*math.Zr, 3)
	for i := 0; i < 2; i++ {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := 0; i < 3; i++ {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := "ABC"
	inValues[0] = c.NewZrFromInt(100)
	inValues[1] = c.NewZrFromInt(50)
	outValues[0] = c.NewZrFromInt(75)
	outValues[1] = c.NewZrFromInt(35)
	outValues[2] = c.NewZrFromInt(40)

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp, c)

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

func prepareInputsOutputs(inValues, outValues, inBF, outBF []*math.Zr, ttype string, pp []*math.G1, c *math.Curve) ([]*math.G1, []*math.G1) {
	inputs := make([]*math.G1, len(inValues))
	outputs := make([]*math.G1, len(outValues))

	for i := 0; i < len(inputs); i++ {
		inputs[i] = c.NewG1()
		inputs[i].Add(pp[0].Mul(c.HashToZr([]byte(ttype))))
		inputs[i].Add(pp[1].Mul(inValues[i]))
		inputs[i].Add(pp[2].Mul(inBF[i]))

	}

	for i := 0; i < len(outputs); i++ {
		outputs[i] = c.NewG1()
		outputs[i].Add(pp[0].Mul(c.HashToZr([]byte(ttype))))
		outputs[i].Add(pp[1].Mul(outValues[i]))
		outputs[i].Add(pp[2].Mul(outBF[i]))

	}
	return inputs, outputs
}

func prepareToken(value *math.Zr, rand *math.Zr, ttype string, pp []*math.G1, c *math.Curve) *math.G1 {
	token := c.NewG1()
	token.Add(pp[0].Mul(c.HashToZr([]byte(ttype))))
	token.Add(pp[1].Mul(value))
	token.Add(pp[2].Mul(rand))
	return token
}
