/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer_test

import (
	"runtime"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/benchmark"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

var (
	TestBits  = uint64(32)
	TestCurve = math2.BLS12_381_BBS_GURVY_FAST_RNG
)

func TestTransfer(t *testing.T) {
	t.Run("parameters and witness are initialized correctly", func(t *testing.T) {
		env, err := newTransferEnv(t)
		require.NoError(t, err)
		proof, err := env.prover.Prove()
		require.NoError(t, err)
		require.NotNil(t, proof)
		err = env.verifier.Verify(proof)
		require.NoError(t, err)
	})
	t.Run("Output Values > Input Values", func(t *testing.T) {
		env, err := newTransferEnvWithWrongSum()
		require.NoError(t, err)

		proof, err := env.prover.Prove()
		require.NoError(t, err)
		require.NotNil(t, proof)
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid transfer proof: invalid sum and type proof")
	})
	t.Run("Output Values out of range", func(t *testing.T) {
		env, err := newTransferEnvWithInvalidRange()
		require.NoError(t, err)

		proof, err := env.prover.Prove()
		require.NotNil(t, proof)
		require.NoError(t, err)
		err = env.verifier.Verify(proof)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid range proof at index 0: invalid range proof")
	})
}

// BenchmarkTransferProofGeneration benchmarks the ZK proof generation for a transfer operation
func BenchmarkTransferProofGeneration(b *testing.B) {
	bits, err := benchmark.Bits(32, 64)
	require.NoError(b, err)
	curves := benchmark.Curves(math.BN254, math.BLS12_381_BBS_GURVY, math2.BLS12_381_BBS_GURVY_FAST_RNG)
	inputs, err := benchmark.NumInputs(1, 2, 3)
	require.NoError(b, err)
	outputs, err := benchmark.NumOutputs(1, 2, 3)
	require.NoError(b, err)
	testCases := benchmark.GenerateCases(bits, curves, inputs, outputs, []int{1})

	for _, tc := range testCases {
		b.Run(tc.Name, func(b *testing.B) {
			// prepare env
			env, err := newBenchmarkTransferEnv(b.N, tc.BenchmarkCase)
			require.NoError(b, err)

			b.ResetTimer()

			i := 0
			for b.Loop() {
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
				i++
			}
		})
	}
}

// TestParallelBenchmarkTransferProofGeneration benchmarks ZK proof generation for a transfer operation when multiple go routines are doing the same thing.
func TestParallelBenchmarkTransferProofGeneration(t *testing.T) {
	bits, err := benchmark.Bits(32)
	require.NoError(t, err)
	curves := benchmark.Curves(math.BN254)
	inputs, err := benchmark.NumInputs(2)
	require.NoError(t, err)
	outputs, err := benchmark.NumOutputs(2)
	require.NoError(t, err)
	workers, err := benchmark.Workers(runtime.NumCPU())
	require.NoError(t, err)
	testCases := benchmark.GenerateCases(bits, curves, inputs, outputs, workers)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			r := benchmark.RunBenchmark(
				tc.BenchmarkCase.Workers,
				benchmark.Duration(),
				func() *benchmarkTransferEnv {
					env, err := newBenchmarkTransferEnv(1, tc.BenchmarkCase)
					require.NoError(t, err)
					return env
				},
				func(env *benchmarkTransferEnv) {
					prover, err := transfer.NewProver(
						env.ProverEnvs[0].a,
						env.ProverEnvs[0].b,
						env.ProverEnvs[0].c,
						env.ProverEnvs[0].d,
						env.pp,
					)
					require.NoError(t, err)
					_, err = prover.Prove()
					require.NoError(t, err)
				},
			)
			r.Print()
		})
	}
}

func setup(bits uint64, curveID math.CurveID) (*v1.PublicParams, error) {
	pp, err := v1.Setup(bits, nil, curveID)
	if err != nil {
		return nil, err
	}
	return pp, nil
}

func prepareZKTransfer() (*transfer.Prover, *transfer.Verifier, error) {
	pp, err := setup(TestBits, TestCurve)
	if err != nil {
		return nil, nil, err
	}

	intw, outtw, in, out, err := prepareInputsForZKTransfer(pp, 2, 2)
	if err != nil {
		return nil, nil, err
	}

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	if err != nil {
		return nil, nil, err
	}
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier, nil
}

func prepareZKTransferWithWrongSum() (*transfer.Prover, *transfer.Verifier, error) {
	pp, err := setup(TestBits, TestCurve)
	if err != nil {
		return nil, nil, err
	}

	intw, outtw, in, out, err := prepareInvalidInputsForZKTransfer(pp)
	if err != nil {
		return nil, nil, err
	}

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	if err != nil {
		return nil, nil, err
	}
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier, nil
}

func prepareZKTransferWithInvalidRange() (*transfer.Prover, *transfer.Verifier, error) {
	pp, err := setup(8, TestCurve)
	if err != nil {
		return nil, nil, err
	}

	intw, outtw, in, out, err := prepareInputsForZKTransfer(pp, 2, 2)
	if err != nil {
		return nil, nil, err
	}

	prover, err := transfer.NewProver(intw, outtw, in, out, pp)
	if err != nil {
		return nil, nil, err
	}
	verifier := transfer.NewVerifier(in, out, pp)
	return prover, verifier, nil
}

func prepareInputsForZKTransfer(pp *v1.PublicParams, numInputs int, numOutputs int) ([]*token.Metadata, []*token.Metadata, []*math.G1, []*math.G1, error) {
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	inBF := make([]*math.Zr, numInputs)
	outBF := make([]*math.Zr, numOutputs)
	inValues := make([]uint64, numInputs)
	outValues := make([]uint64, numOutputs)
	for i := range numInputs {
		inBF[i] = c.NewRandomZr(rand)
	}
	for i := range numOutputs {
		outBF[i] = c.NewRandomZr(rand)
	}
	ttype := token2.Type("ABC")

	// prepare inputs
	sumInputs := uint64(0)
	for i := range numInputs {
		v := uint64(i*10 + 500)
		sumInputs += v
		inValues[i] = v
	}

	outputValue := sumInputs / uint64(numOutputs)
	sumOutputs := uint64(0)
	for i := range numOutputs {
		outValues[i] = outputValue
		sumOutputs += outputValue
	}
	// add any adjustment to the last output
	delta := sumInputs - sumOutputs
	if delta > 0 {
		outValues[0] += delta
	}

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.PedersenGenerators, c)
	intw := make([]*token.Metadata, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.Metadata{BlindingFactor: inBF[i], Value: c.NewZrFromUint64(inValues[i]), Type: ttype}
	}

	outtw := make([]*token.Metadata, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.Metadata{BlindingFactor: outBF[i], Value: c.NewZrFromUint64(outValues[i]), Type: ttype}
	}

	return intw, outtw, in, out, nil
}

func prepareInvalidInputsForZKTransfer(pp *v1.PublicParams) ([]*token.Metadata, []*token.Metadata, []*math.G1, []*math.G1, error) {
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	if err != nil {
		return nil, nil, nil, nil, err
	}

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

	return intw, outtw, in, out, nil
}

type transferEnv struct {
	prover   *transfer.Prover
	verifier *transfer.Verifier
}

func newTransferEnv(tb testing.TB) (*transferEnv, error) {
	tb.Helper()
	prover, verifier, err := prepareZKTransfer()
	if err != nil {
		return nil, err
	}
	return &transferEnv{
		prover:   prover,
		verifier: verifier,
	}, nil
}

func newTransferEnvWithWrongSum() (*transferEnv, error) {
	prover, verifier, err := prepareZKTransferWithWrongSum()
	if err != nil {
		return nil, err
	}
	return &transferEnv{
		prover:   prover,
		verifier: verifier,
	}, nil
}

func newTransferEnvWithInvalidRange() (*transferEnv, error) {
	prover, verifier, err := prepareZKTransferWithInvalidRange()
	if err != nil {
		return nil, err
	}
	return &transferEnv{
		prover:   prover,
		verifier: verifier,
	}, nil
}

type singleProverEnv struct {
	a []*token.Metadata
	b []*token.Metadata
	c []*math.G1
	d []*math.G1
}

type benchmarkTransferEnv struct {
	ProverEnvs []singleProverEnv
	pp         *v1.PublicParams
}

func newBenchmarkTransferEnv(n int, benchmarkCase *benchmark.Case) (*benchmarkTransferEnv, error) {
	pp, err := setup(benchmarkCase.Bits, benchmarkCase.CurveID)
	if err != nil {
		return nil, err
	}

	entries := make([]singleProverEnv, n)
	for i := 0; i < n; i++ {
		intw, outtw, in, out, err := prepareInputsForZKTransfer(pp, benchmarkCase.NumInputs, benchmarkCase.NumOutputs)
		if err != nil {
			return nil, err
		}
		entries[i] = singleProverEnv{
			a: intw,
			b: outtw,
			c: in,
			d: out,
		}
	}
	return &benchmarkTransferEnv{ProverEnvs: entries, pp: pp}, nil
}
