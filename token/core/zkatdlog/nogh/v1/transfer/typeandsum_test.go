/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Input/Output well formedness", func() {
	var (
		iow      *transfer.TypeAndSumWitness
		pp       []*math.G1
		verifier *transfer.TypeAndSumVerifier
		prover   *transfer.TypeAndSumProver
		in       []*math.G1
		out      []*math.G1
		inBF     []*math.Zr
		outBF    []*math.Zr
		c        *math.Curve
		com      *math.G1
		//		parallelism = 100
	)
	BeforeEach(func() {
		c = math.Curves[1]
		pp = preparePedersenParameters(c)
		iow, in, out, inBF, outBF, com = prepareIOCProver(pp, c)
		prover = transfer.NewTypeAndSumProver(iow, pp, in, out, com, c)
		verifier = transfer.NewTypeAndSumVerifier(pp, in, out, c)
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				Expect(proof.Challenge).NotTo(BeNil())
				Expect(proof.EqualityOfSum).NotTo(BeNil())
				Expect(proof.Type).NotTo(BeNil())
				Expect(proof.InputBlindingFactors).To(HaveLen(2))
				Expect(proof.InputValues).To(HaveLen(2))
			})
		})
	})
	Describe("Verify", func() {
		BeforeEach(func() {
			prover = transfer.NewTypeAndSumProver(iow, pp, in, out, com, c)
			verifier = transfer.NewTypeAndSumVerifier(pp, in, out, c)
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
		/*Context("The proof is generated and verified in parallel", func() {
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
		})*/

		Context("The proof is not generated correctly: wrong type", func() {
			It("fails", func() {
				// change type encoded in the commitments
				token := prepareToken(c.NewZrFromInt(100), inBF[0], "XYZ", pp, c)
				// prover assumed to guess the type (e.g. ABC)
				prover.Inputs[0] = token
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verification fails
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid sum and type proof"))
			})
		})
		Context("The proof is not generated correctly: wrong Values", func() {
			It("fails", func() {
				// change the value encoded in the input commitment
				token := prepareToken(c.NewZrFromInt(80), inBF[0], "ABC", pp, c)
				// prover guess the value of the committed Values (e.g. 100)
				prover.Inputs[0] = token
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verification fails
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid sum and type proof"))
			})
		})
		Context("The proof is not generated correctly: input sum != output sums", func() {
			It("fails", func() {
				// prover wants to increase the value of the output out of the blue
				token := prepareToken(c.NewZrFromInt(90), outBF[0], "ABC", pp, c)
				// prover generates a proof
				prover.Outputs[0] = token
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				// verification should fail
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("invalid sum and type proof"))
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
				Expect(err.Error()).To(Equal("invalid sum and type proof"))
			})
		})
	})
})

func preparePedersenParameters(c *math.Curve) []*math.G1 {
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	pp := make([]*math.G1, 3)

	for i := range 3 {
		pp[i] = c.GenG1.Mul(c.NewRandomZr(rand))
	}
	return pp
}

func prepareIOCProver(pp []*math.G1, c *math.Curve) (*transfer.TypeAndSumWitness, []*math.G1, []*math.G1, []*math.Zr, []*math.Zr, *math.G1) {
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 3)
	inValues := make([]uint64, 2)
	outValues := make([]uint64, 3)
	for i := range 2 {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := range 3 {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := token2.Type("ABC")
	inValues[0] = 100
	inValues[1] = 50
	outValues[0] = 75
	outValues[1] = 35
	outValues[2] = 40

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp, c)

	intw := make([]*token.Metadata, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.Metadata{BlindingFactor: inBF[i], Value: c.NewZrFromUint64(inValues[i]), Type: ttype}
	}

	outtw := make([]*token.Metadata, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.Metadata{BlindingFactor: outBF[i], Value: c.NewZrFromUint64(outValues[i]), Type: ttype}
	}
	typeBlindingFactor := c.NewRandomZr(rand)
	commitmentToType := pp[0].Mul(c.HashToZr([]byte(ttype)))
	commitmentToType.Add(pp[2].Mul(typeBlindingFactor))

	return transfer.NewTypeAndSumWitness(typeBlindingFactor, intw, outtw, c), in, out, inBF, outBF, commitmentToType
}

func prepareInputsOutputs(inValues, outValues []uint64, inBF, outBF []*math.Zr, ttype token2.Type, pp []*math.G1, c *math.Curve) ([]*math.G1, []*math.G1) {
	inputs := make([]*math.G1, len(inValues))
	outputs := make([]*math.G1, len(outValues))

	for i := 0; i < len(inputs); i++ {
		inputs[i] = pp[0].Mul(c.HashToZr([]byte(ttype)))
		inputs[i].Add(pp[1].Mul(c.NewZrFromInt(int64(inValues[i]))))
		inputs[i].Add(pp[2].Mul(inBF[i]))
	}

	for i := 0; i < len(outputs); i++ {
		outputs[i] = pp[0].Mul(c.HashToZr([]byte(ttype)))
		outputs[i].Add(pp[1].Mul(c.NewZrFromInt(int64(outValues[i]))))
		outputs[i].Add(pp[2].Mul(outBF[i]))
	}
	return inputs, outputs
}

func prepareToken(value *math.Zr, rand *math.Zr, ttype string, pp []*math.G1, c *math.Curve) *math.G1 {
	token := pp[0].Mul(c.HashToZr([]byte(ttype)))
	token.Add(pp[1].Mul(value))
	token.Add(pp[2].Mul(rand))
	return token
}
