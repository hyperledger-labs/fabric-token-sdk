/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

type TypeAndSumEnv struct {
	prover   *transfer.TypeAndSumProver
	verifier *transfer.TypeAndSumVerifier
	c        *math.Curve
	inBF     []*math.Zr
	pp       []*math.G1
	outBF    []*math.Zr
}

func NewTypeAndSumEnv(t *testing.T) *TypeAndSumEnv {
	t.Helper()
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
	c = math.Curves[1]
	pp = preparePedersenParameters(t, c)
	iow, in, out, inBF, outBF, com = prepareIOCProver(t, pp, c)
	prover = transfer.NewTypeAndSumProver(iow, pp, in, out, com, c)
	verifier = transfer.NewTypeAndSumVerifier(pp, in, out, c)

	return &TypeAndSumEnv{
		c:        c,
		prover:   prover,
		verifier: verifier,
		inBF:     inBF,
		pp:       pp,
		outBF:    outBF,
	}
}

func TestTypeAndSum(t *testing.T) {
	t.Run("parameters and witness are initialized correctly", func(t *testing.T) {
		env := NewTypeAndSumEnv(t)
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		require.NotNil(t, proof)
		require.NotNil(t, proof.Challenge)
		require.NotNil(t, proof.EqualityOfSum)
		require.NotNil(t, proof.Type)
		require.Len(t, proof.InputBlindingFactors, 2)
		require.Len(t, proof.InputValues, 2)
	})
	t.Run("the proof is generated honestly", func(t *testing.T) {
		env := NewTypeAndSumEnv(t)
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		// verify
		err = env.verifier.Verify(proof)
		require.NoError(t, err)
	})

	t.Run("The proof is not generated correctly: wrong type", func(t *testing.T) {
		env := NewTypeAndSumEnv(t)
		// change type encoded in the commitments
		token := prepareToken(env.c.NewZrFromInt(100), env.inBF[0], "XYZ", env.pp, env.c)
		// prover assumed to guess the type (e.g. ABC)
		env.prover.Inputs[0] = token
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		// verification fails
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.EqualError(t, err, "invalid sum and type proof")
	})
	t.Run("The proof is not generated correctly: wrong Values", func(t *testing.T) {
		env := NewTypeAndSumEnv(t)
		// change the value encoded in the input commitment
		token := prepareToken(env.c.NewZrFromInt(80), env.inBF[0], "ABC", env.pp, env.c)
		// prover guess the value of the committed Values (e.g. 100)
		env.prover.Inputs[0] = token
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		// verification fails
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.EqualError(t, err, "invalid sum and type proof")
	})
	t.Run("The proof is not generated correctly: input sum != output sums", func(t *testing.T) {
		env := NewTypeAndSumEnv(t)
		// prover wants to increase the value of the output out of the blue
		token := prepareToken(env.c.NewZrFromInt(90), env.outBF[0], "ABC", env.pp, env.c)
		// prover generates a proof
		env.prover.Outputs[0] = token
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		// verification should fail
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.EqualError(t, err, "invalid sum and type proof")
	})
	t.Run("The proof is not generated correctly: wrong blindingFactors", func(t *testing.T) {
		env := NewTypeAndSumEnv(t)
		// prover guess the blindingFactors
		rand, err := env.c.Rand()
		require.NoError(t, err)
		token := prepareToken(env.c.NewZrFromInt(100), env.c.NewRandomZr(rand), "ABC", env.pp, env.c)
		env.verifier.Inputs[0] = token
		// prover generate proof
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		// verification fails
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.EqualError(t, err, "invalid sum and type proof")
	})
}

func preparePedersenParameters(tb testing.TB, c *math.Curve) []*math.G1 {
	tb.Helper()
	rand, err := c.Rand()
	require.NoError(tb, err)

	pp := make([]*math.G1, 3)

	for i := range 3 {
		pp[i] = c.GenG1.Mul(c.NewRandomZr(rand))
	}
	return pp
}

func prepareIOCProver(tb testing.TB, pp []*math.G1, c *math.Curve) (*transfer.TypeAndSumWitness, []*math.G1, []*math.G1, []*math.Zr, []*math.Zr, *math.G1) {
	tb.Helper()
	rand, err := c.Rand()
	require.NoError(tb, err)

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
		inputs[i].Add(pp[1].Mul(c.NewZrFromInt(int64(inValues[i])))) // #nosec G115
		inputs[i].Add(pp[2].Mul(inBF[i]))
	}

	for i := 0; i < len(outputs); i++ {
		outputs[i] = pp[0].Mul(c.HashToZr([]byte(ttype)))
		outputs[i].Add(pp[1].Mul(c.NewZrFromInt(int64(outValues[i])))) // #nosec G115
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
