/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/transfer"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
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
}*/

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
				Expect(err.Error()).To(ContainSubstring("invalid transfer proof: invalid sum and type proof"))
			})
		})
		Context("Output Values out of range", func() {
			BeforeEach(func() {
				prover, verifier = prepareZKTransferWithInvalidRange()
			})
			It("fails during proof generation", func() {
				proof, err := prover.Prove()
				Expect(proof).NotTo(BeNil())
				Expect(err).NotTo(HaveOccurred())
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid range proof at index 0: invalid range proof"))
			})
		})
	})
})

func prepareZKTransfer() (*transfer.Prover, *transfer.Verifier) {
	pp, err := v1.Setup(32, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	intw, outtw, in, out := prepareInputsForZKTransfer(pp)

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	Expect(err).NotTo(HaveOccurred())
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithWrongSum() (*transfer.Prover, *transfer.Verifier) {
	pp, err := v1.Setup(32, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	intw, outtw, in, out := prepareInvalidInputsForZKTransfer(pp)

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	Expect(err).NotTo(HaveOccurred())
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithInvalidRange() (*transfer.Prover, *transfer.Verifier) {
	pp, err := v1.Setup(8, nil, math.FP256BN_AMCL)
	Expect(err).NotTo(HaveOccurred())

	intw, outtw, in, out := prepareInputsForZKTransfer(pp)

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)
	Expect(err).NotTo(HaveOccurred())
	return prover, verifier
}

func prepareInputsForZKTransfer(pp *v1.PublicParams) ([]*token.TokenDataWitness, []*token.TokenDataWitness, []*math.G1, []*math.G1) {
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 2)
	inValues := make([]uint64, 2)
	outValues := make([]uint64, 2)
	for i := 0; i < 2; i++ {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := 0; i < 2; i++ {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := token2.Type("ABC")
	inValues[0] = 220
	inValues[1] = 60
	outValues[0] = 260
	outValues[1] = 20

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.PedersenGenerators, c)
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	return intw, outtw, in, out
}

func prepareInvalidInputsForZKTransfer(pp *v1.PublicParams) ([]*token.TokenDataWitness, []*token.TokenDataWitness, []*math.G1, []*math.G1) {
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 2)
	inValues := make([]uint64, 2)
	outValues := make([]uint64, 2)
	for i := 0; i < 2; i++ {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := 0; i < 2; i++ {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := token2.Type("ABC")
	inValues[0] = 90
	inValues[1] = 60
	outValues[0] = 110
	outValues[1] = 45

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.PedersenGenerators, c)
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	return intw, outtw, in, out
}
