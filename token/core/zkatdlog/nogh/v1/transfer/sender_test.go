/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer_test

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/benchmark"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSender(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env, err := newSenderEnv(nil, 3, 2)
		require.NoError(t, err)

		transfer, _, err := env.sender.GenerateZKTransfer(t.Context(), env.outvalues, env.owners)
		require.NoError(t, err)
		assert.NotNil(t, transfer)
		raw, err := transfer.Serialize()
		require.NoError(t, err)

		sig, err := env.sender.SignTokenActions(raw)
		require.NoError(t, err)
		assert.Equal(t, 3, env.fakeSigningIdentity.SignCallCount())
		assert.Len(t, sig, 3)
	})

	t.Run("when signature fails", func(t *testing.T) {
		env, err := newSenderEnv(nil, 3, 2)
		require.NoError(t, err)
		env.fakeSigningIdentity.SignReturnsOnCall(2, nil, errors.New("banana republic"))
		transfer, _, err := env.sender.GenerateZKTransfer(t.Context(), env.outvalues, env.owners)
		require.NoError(t, err)
		assert.NotNil(t, transfer)
		raw, err := transfer.Serialize()
		require.NoError(t, err)

		sig, err := env.sender.SignTokenActions(raw)
		require.Error(t, err)
		assert.Nil(t, sig)
		assert.Equal(t, 3, env.fakeSigningIdentity.SignCallCount())
		assert.Contains(t, err.Error(), "banana republic")
	})
}

// BenchmarkSender benchmarks transfer action generation and serialization.
// This includes the proof generation as well.
func BenchmarkSender(b *testing.B) {
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
			env, err := newBenchmarkSenderEnv(b.N, tc.BenchmarkCase)
			require.NoError(b, err)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				transfer, _, err := env.SenderEnvs[i].sender.GenerateZKTransfer(
					b.Context(),
					env.SenderEnvs[i].outvalues,
					env.SenderEnvs[i].owners,
				)
				require.NoError(b, err)
				assert.NotNil(b, transfer)
				_, err = transfer.Serialize()
				require.NoError(b, err)
			}
		})
	}
}

// TestParallelBenchmarkSender benchmarks transfer action generation and serialization when multiple go routines are doing the same thing.
func TestParallelBenchmarkSender(t *testing.T) {
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
				func() *benchmarkSenderEnv {
					env, err := newBenchmarkSenderEnv(1, tc.BenchmarkCase)
					require.NoError(t, err)
					return env
				},
				func(env *benchmarkSenderEnv) {
					transfer, _, err := env.SenderEnvs[0].sender.GenerateZKTransfer(
						t.Context(),
						env.SenderEnvs[0].outvalues,
						env.SenderEnvs[0].owners,
					)
					require.NoError(t, err)
					assert.NotNil(t, transfer)
					_, err = transfer.Serialize()
					require.NoError(t, err)
				},
			)
			r.Print()
		})
	}
}

// BenchmarkVerificationSenderProof benchmarks transfer action deserialization and proof verification.
func BenchmarkVerificationSenderProof(b *testing.B) {
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
			env, err := newBenchmarkSenderProofVerificationEnv(b.Context(), b.N, tc.BenchmarkCase)
			require.NoError(b, err)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// deserialize action
				ta := &transfer.Action{}
				require.NoError(b, ta.Deserialize(env.SenderEnvs[i].transferRaw))
				inputTokens := make([]*math.G1, len(ta.Inputs))
				for j, in := range ta.Inputs {
					inputTokens[j] = in.Token.Data
				}

				// instantiate the verifier and verify
				require.NoError(b,
					transfer.NewVerifier(
						inputTokens,
						ta.GetOutputCommitments(),
						env.SenderEnvs[i].sender.PublicParams,
					).Verify(ta.GetProof()),
				)
			}
		})
	}
}

// TestParallelBenchmarkVerificationSenderProof benchmarks transfer action deserialization and proof verification when multiple go routines are doing the same thing.
func TestParallelBenchmarkVerificationSenderProof(t *testing.T) {
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
				func() *benchmarkSenderEnv {
					env, err := newBenchmarkSenderProofVerificationEnv(t.Context(), 1, tc.BenchmarkCase)
					require.NoError(t, err)
					return env
				},
				func(env *benchmarkSenderEnv) {
					// deserialize action
					ta := &transfer.Action{}
					require.NoError(t, ta.Deserialize(env.SenderEnvs[0].transferRaw))
					inputTokens := make([]*math.G1, len(ta.Inputs))
					for j, in := range ta.Inputs {
						inputTokens[j] = in.Token.Data
					}

					// instantiate the verifier and verify
					require.NoError(t,
						transfer.NewVerifier(
							inputTokens,
							ta.GetOutputCommitments(),
							env.SenderEnvs[0].sender.PublicParams,
						).Verify(ta.GetProof()),
					)
				},
			)
			r.Print()
		})
	}
}

func prepareTokens(values, bf []*math.Zr, ttype string, pp []*math.G1, curve *math.Curve) []*math.G1 {
	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = prepareToken(values[i], bf[i], ttype, pp, curve)
	}
	return tokens
}

type senderEnv struct {
	sender              *transfer.Sender
	outvalues           []uint64
	owners              [][]byte
	fakeSigningIdentity *mock.SigningIdentity
	transferRaw         []byte
}

func newSenderEnv(pp *v1.PublicParams, numInputs int, numOutputs int) (*senderEnv, error) {
	var (
		fakeSigningIdentity *mock.SigningIdentity
		signers             []driver.Signer

		sender *transfer.Sender

		invalues  []*math.Zr
		outvalues []uint64
		inBF      []*math.Zr
		tokens    []*token.Token

		owners [][]byte
		ids    []*token2.ID
	)
	var err error
	if pp == nil {
		pp, err = setup(TestBits, TestCurve)
		if err != nil {
			return nil, err
		}
	}
	signers = make([]driver.Signer, numInputs)
	fakeSigningIdentity = &mock.SigningIdentity{}
	invalues = make([]*math.Zr, numInputs)
	c := math.Curves[pp.Curve]
	inBF = make([]*math.Zr, numInputs)
	ids = make([]*token2.ID, numInputs)
	rand, err := c.Rand()
	if err != nil {
		return nil, err
	}
	tokens = make([]*token.Token, numInputs)
	inputInf := make([]*token.Metadata, numInputs)

	owners = make([][]byte, numOutputs)
	outvalues = make([]uint64, numOutputs)

	// prepare inputs
	sumInputs := int64(0)
	for i := range numInputs {
		signers[i] = fakeSigningIdentity
		fakeSigningIdentity.SignReturnsOnCall(i, []byte(fmt.Sprintf("signer[%d]", i)), nil)
		v := int64(i*10 + 10)
		sumInputs += v
		invalues[i] = c.NewZrFromInt(v)
		inBF[i] = c.NewRandomZr(rand)
		ids[i] = &token2.ID{TxId: strconv.Itoa(i)}
	}
	inputs := prepareTokens(invalues, inBF, "ABC", pp.PedersenGenerators, c)

	for i := range numInputs {
		tokens[i] = &token.Token{Data: inputs[i], Owner: []byte(fmt.Sprintf("alice-%d", i))}
		inputInf[i] = &token.Metadata{Type: "ABC", Value: invalues[i], BlindingFactor: inBF[i]}
	}

	outputValue := uint64(sumInputs / int64(numOutputs))
	sumOutputs := int64(0)
	for i := range numOutputs {
		owners[i] = []byte("bob")
		outvalues[i] = outputValue
		sumOutputs += int64(outputValue)
	}
	// add any adjustment to the last output
	delta := sumInputs - sumOutputs
	if delta > 0 {
		outvalues[0] += uint64(delta)
	}

	sender, err = transfer.NewSender(signers, tokens, ids, inputInf, pp)
	if err != nil {
		return nil, err
	}

	return &senderEnv{
		sender:              sender,
		outvalues:           outvalues,
		owners:              owners,
		fakeSigningIdentity: fakeSigningIdentity,
	}, nil
}

type benchmarkSenderEnv struct {
	SenderEnvs []*senderEnv
}

func newBenchmarkSenderEnv(n int, benchmarkCase *benchmark.Case) (*benchmarkSenderEnv, error) {
	envs := make([]*senderEnv, n)
	pp, err := setup(benchmarkCase.Bits, benchmarkCase.CurveID)
	if err != nil {
		return nil, err
	}
	for i := range envs {
		envs[i], err = newSenderEnv(pp, benchmarkCase.NumInputs, benchmarkCase.NumOutputs)
		if err != nil {
			return nil, err
		}
	}
	return &benchmarkSenderEnv{SenderEnvs: envs}, nil
}

func newBenchmarkSenderProofVerificationEnv(ctx context.Context, n int, benchmarkCase *benchmark.Case) (*benchmarkSenderEnv, error) {
	envs := make([]*senderEnv, n)
	pp, err := setup(benchmarkCase.Bits, benchmarkCase.CurveID)
	if err != nil {
		return nil, err
	}
	for i := range envs {
		env, err := newSenderEnv(pp, benchmarkCase.NumInputs, benchmarkCase.NumOutputs)
		if err != nil {
			return nil, err
		}
		transfer, _, err := env.sender.GenerateZKTransfer(
			ctx,
			env.outvalues,
			env.owners,
		)
		if err != nil {
			return nil, err
		}
		raw, err := transfer.Serialize()
		if err != nil {
			return nil, err
		}

		env.transferRaw = raw
		envs[i] = env
	}
	return &benchmarkSenderEnv{SenderEnvs: envs}, nil
}
