/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"testing"

	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

const TestCurve = math.BN254

type TransferEnv struct {
	prover   *transfer.Prover
	verifier *transfer.Verifier
}

func NewTransferEnv(tb testing.TB) *TransferEnv {
	tb.Helper()
	prover, verifier := prepareZKTransfer(tb)
	return &TransferEnv{
		prover:   prover,
		verifier: verifier,
	}
}

func NewTransferEnvWithWrongSum(tb testing.TB) *TransferEnv {
	tb.Helper()
	prover, verifier := prepareZKTransferWithWrongSum(tb)
	return &TransferEnv{
		prover:   prover,
		verifier: verifier,
	}
}

func NewTransferEnvWithInvalidRange(tb testing.TB) *TransferEnv {
	tb.Helper()
	prover, verifier := prepareZKTransferWithInvalidRange(tb)
	return &TransferEnv{
		prover:   prover,
		verifier: verifier,
	}
}

func TestTransfer(t *testing.T) {
	t.Run("parameters and witness are initialized correctly", func(t *testing.T) {
		env := NewTransferEnv(t)
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		require.NotNil(t, proof)
		err = env.verifier.Verify(proof)
		require.NoError(t, err)
	})
	t.Run("Output Values > Input Values", func(t *testing.T) {
		env := NewTransferEnvWithWrongSum(t)

		proof, err := env.prover.Prove()
		require.NoError(t, err)
		require.NotNil(t, proof)
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid transfer proof: invalid sum and type proof")
	})
	t.Run("Output Values out of range", func(t *testing.T) {
		env := NewTransferEnvWithInvalidRange(t)
		proof, err := env.prover.Prove()
		require.NotNil(t, proof)
		require.NoError(t, err)
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid range proof at index 0: invalid range proof")
	})
}

func prepareZKTransfer(tb testing.TB) (*transfer.Prover, *transfer.Verifier) {
	tb.Helper()
	pp, err := v1.Setup(32, nil, TestCurve)
	require.NoError(tb, err)

	intw, outtw, in, out := prepareInputsForZKTransfer(tb, pp)

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	require.NoError(tb, err)
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithWrongSum(tb testing.TB) (*transfer.Prover, *transfer.Verifier) {
	tb.Helper()
	pp, err := v1.Setup(32, nil, TestCurve)
	require.NoError(tb, err)

	intw, outtw, in, out := prepareInvalidInputsForZKTransfer(tb, pp)

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	require.NoError(tb, err)
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithInvalidRange(tb testing.TB) (*transfer.Prover, *transfer.Verifier) {
	tb.Helper()
	pp, err := v1.Setup(8, nil, TestCurve)
	require.NoError(tb, err)

	intw, outtw, in, out := prepareInputsForZKTransfer(tb, pp)

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)
	require.NoError(tb, err)
	return prover, verifier
}

func prepareInputsForZKTransfer(tb testing.TB, pp *v1.PublicParams) ([]*token.Metadata, []*token.Metadata, []*math.G1, []*math.G1) {
	tb.Helper()
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	require.NoError(tb, err)

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 2)
	inValues := make([]uint64, 2)
	outValues := make([]uint64, 2)
	for i := range 2 {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := range 2 {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := token2.Type("ABC")
	inValues[0] = 220
	inValues[1] = 60
	outValues[0] = 260
	outValues[1] = 20

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.PedersenGenerators, c)
	intw := make([]*token.Metadata, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.Metadata{BlindingFactor: inBF[i], Value: c.NewZrFromUint64(inValues[i]), Type: ttype}
	}

	outtw := make([]*token.Metadata, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.Metadata{BlindingFactor: outBF[i], Value: c.NewZrFromUint64(outValues[i]), Type: ttype}
	}

	return intw, outtw, in, out
}

func prepareInvalidInputsForZKTransfer(tb testing.TB, pp *v1.PublicParams) ([]*token.Metadata, []*token.Metadata, []*math.G1, []*math.G1) {
	tb.Helper()

	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	require.NoError(tb, err)

	inBF := make([]*math.Zr, 2)
	outBF := make([]*math.Zr, 2)
	inValues := make([]uint64, 2)
	outValues := make([]uint64, 2)
	for i := range 2 {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := range 2 {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := token2.Type("ABC")
	inValues[0] = 90
	inValues[1] = 60
	outValues[0] = 110
	outValues[1] = 45

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.PedersenGenerators, c)
	intw := make([]*token.Metadata, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.Metadata{BlindingFactor: inBF[i], Value: c.NewZrFromUint64(inValues[i]), Type: ttype}
	}

	outtw := make([]*token.Metadata, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.Metadata{BlindingFactor: outBF[i], Value: c.NewZrFromUint64(outValues[i]), Type: ttype}
	}

	return intw, outtw, in, out
}

type SingleProverEnv struct {
	a []*token.Metadata
	b []*token.Metadata
	c []*math.G1
	d []*math.G1
}

type BenchmarkTransferEnv struct {
	ProverEnvs []SingleProverEnv
	pp         *v1.PublicParams
}

func NewBenchmarkTransferEnv(tb testing.TB, n int) *BenchmarkTransferEnv {
	tb.Helper()
	pp, err := v1.Setup(32, nil, TestCurve)
	require.NoError(tb, err)

	entries := make([]SingleProverEnv, n)
	for i := 0; i < n; i++ {
		intw, outtw, in, out := prepareInputsForZKTransfer(tb, pp)
		entries[i] = SingleProverEnv{
			a: intw,
			b: outtw,
			c: in,
			d: out,
		}
	}
	return &BenchmarkTransferEnv{ProverEnvs: entries, pp: pp}
}

func BenchmarkTransfer(b *testing.B) {
	b.ReportAllocs()

	// prepare env
	env := NewBenchmarkTransferEnv(b, b.N)

	// Optional: Reset timer if you had expensive setup code above
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		prover, err := transfer.NewProver(
			env.ProverEnvs[i].a,
			env.ProverEnvs[i].b,
			env.ProverEnvs[i].c,
			env.ProverEnvs[i].d,
			env.pp,
		)
		require.NoError(b, err)
		_, err = prover.Prove()
		require.NoError(b, err)
	}
}
