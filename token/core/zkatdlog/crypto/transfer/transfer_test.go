/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"sync"
	"testing"

	math "github.com/IBM/mathlib"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
)

func TestParallelProveVerify(t *testing.T) {
	parallelism := 1000

	prover, verifier := prepareOwnershipTransfer()

	var wg sync.WaitGroup
	wg.Add(parallelism)

	for i := 0; i < parallelism; i++ {
		go func() {
			defer wg.Done()
			proof, err := prover.Prove()
			Expect(err).NotTo(HaveOccurred())
			Expect(proof).NotTo(BeNil())
			err = verifier.Verify(proof)
			Expect(err).NotTo(HaveOccurred())
		}()
	}

	wg.Wait()
}

var _ = Describe("Transfer", func() {
	var (
		prover   *transfer.Prover
		verifier *transfer.Verifier
	)
	BeforeEach(func() {
		prover, verifier = prepareZKTransfer()
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("Output Values > Input Values", func() {
			BeforeEach(func() {
				prover, verifier = prepareZKTransferWithWrongSum()
			})
			It("fails", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid zero-knowledge transfer"))
			})
		})
		Context("Output Values out of range", func() {
			BeforeEach(func() {
				prover, verifier = prepareZKTransferWithInvalidRange()
			})
			It("fails during proof generation", func() {
				proof, err := prover.Prove()
				Expect(proof).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("can't compute range proof: value of token outside authorized range"))
			})
		})
	})
})

func prepareZKTransfer() (*transfer.Prover, *transfer.Verifier) {
	pp, err := crypto.Setup(100, 2, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	wfw, in, out := prepareInputsForZKTransfer(pp)

	inBF := wfw.GetInBlindingFators()
	outBF := wfw.GetOutBlindingFators()

	inValues := wfw.GetInValues()
	outValues := wfw.GetOutValues()

	ttype := "ABC"
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}
	prover := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithWrongSum() (*transfer.Prover, *transfer.Verifier) {
	pp, err := crypto.Setup(100, 2, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	wfw, in, out := prepareInvalidInputsForZKTransfer(pp)

	inBF := wfw.GetInBlindingFators()
	outBF := wfw.GetOutBlindingFators()

	inValues := wfw.GetInValues()
	outValues := wfw.GetOutValues()

	ttype := "ABC"
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	prover := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithInvalidRange() (*transfer.Prover, *transfer.Verifier) {
	pp, err := crypto.Setup(10, 2, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	wfw, in, out := prepareInputsForZKTransfer(pp)

	inBF := wfw.GetInBlindingFators()
	outBF := wfw.GetOutBlindingFators()

	inValues := wfw.GetInValues()
	outValues := wfw.GetOutValues()

	ttype := "ABC"
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	prover := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)
	return prover, verifier
}

func prepareInputsForZKTransfer(pp *crypto.PublicParams) (*transfer.WellFormednessWitness, []*math.G1, []*math.G1) {
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 2)
	inValues := make([]*math.Zr, 2)
	outValues := make([]*math.Zr, 2)
	for i := 0; i < 2; i++ {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := 0; i < 2; i++ {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := "ABC"
	inValues[0] = c.NewZrFromInt(90)
	inValues[1] = c.NewZrFromInt(60)
	outValues[0] = c.NewZrFromInt(50)
	outValues[1] = c.NewZrFromInt(100)

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.ZKATPedParams, c)
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	return transfer.NewWellFormednessWitness(intw, outtw), in, out
}

func prepareInvalidInputsForZKTransfer(pp *crypto.PublicParams) (*transfer.WellFormednessWitness, []*math.G1, []*math.G1) {
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 2)
	inValues := make([]*math.Zr, 2)
	outValues := make([]*math.Zr, 2)
	for i := 0; i < 2; i++ {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := 0; i < 2; i++ {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := "ABC"
	inValues[0] = c.NewZrFromInt(90)
	inValues[1] = c.NewZrFromInt(60)
	outValues[0] = c.NewZrFromInt(110)
	outValues[1] = c.NewZrFromInt(45)

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.ZKATPedParams, c)
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	return transfer.NewWellFormednessWitness(intw, outtw), in, out
}

func prepareInputsForOwnershipTransfer(pp *crypto.PublicParams) (*transfer.WellFormednessWitness, []*math.G1, []*math.G1) {
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	inBF := c.NewRandomZr(rand)
	outBF := c.NewRandomZr(rand)
	ttype := "ABC"
	inValue := c.NewZrFromInt(90)
	outValue := c.NewZrFromInt(90)

	in, out := prepareInputsOutputs([]*math.Zr{inValue}, []*math.Zr{outValue}, []*math.Zr{inBF}, []*math.Zr{outBF}, ttype, pp.ZKATPedParams, c)
	intw := make([]*token.TokenDataWitness, 1)
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF, Value: inValue, Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, 1)
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF, Value: outValue, Type: ttype}
	}
	return transfer.NewWellFormednessWitness(intw, outtw), in, out
}

func prepareOwnershipTransfer() (*transfer.Prover, *transfer.Verifier) {
	pp, err := crypto.Setup(100, 2, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	wfw, in, out := prepareInputsForOwnershipTransfer(pp)

	inBF := wfw.GetInBlindingFators()
	outBF := wfw.GetOutBlindingFators()

	inValues := wfw.GetInValues()
	outValues := wfw.GetOutValues()

	ttype := "ABC"
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}
	prover := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}
