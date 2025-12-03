/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue_test

import (
	"strconv"
	"testing"

	math "github.com/IBM/mathlib"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue/mock"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProverVerifier(t *testing.T) {
	prover, verifier := prepareZKIssue(t, 32, math.BN254, 2)
	proof, err := prover.Prove()
	assert.NoError(t, err)
	assert.NotNil(t, proof)
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func TestIssuer(t *testing.T) {
	pp := setup(t, 32, math.BN254)
	issuer := issue2.NewIssuer("ABC", &mock.SigningIdentity{}, pp)
	action, _, err := issuer.GenerateZKIssue([]uint64{10, 20}, [][]byte{[]byte("alice"), []byte("bob")})
	require.NoError(t, err)

	// check the proof
	coms := make([]*math.G1, len(action.Outputs))
	for i := range len(action.Outputs) {
		coms[i] = action.Outputs[i].Data
	}
	require.NoError(t, issue2.NewVerifier(coms, pp).Verify(action.GetProof()))
}

func BenchmarkIssuer(b *testing.B) {
	bits, err := benchmark2.Bits(32, 64)
	require.NoError(b, err)
	curves := benchmark2.Curves(math.BN254, math.BLS12_381_BBS_GURVY, math2.BLS12_381_BBS_GURVY_FAST_RNG)
	outputs, err := benchmark2.NumOutputs(1, 2, 3)
	require.NoError(b, err)
	testCases := benchmark2.GenerateCases(bits, curves, nil, outputs, nil)

	for _, tc := range testCases {
		b.Run(tc.Name, func(b *testing.B) {
			env := newBenchmarkIssuerEnv(b, b.N, tc.BenchmarkCase)

			// Optional: Reset timer if you had expensive setup code above
			b.ResetTimer()

			i := 0
			for b.Loop() {
				issuer := issue2.NewIssuer("ABC", &mock.SigningIdentity{}, env.IssuerEnvs[i].pp)
				action, _, err := issuer.GenerateZKIssue(
					env.IssuerEnvs[i].outputValues,
					env.IssuerEnvs[i].outputOwners,
				)
				require.NoError(b, err)
				_, err = action.Serialize()
				require.NoError(b, err)
				i++
			}
		})
	}
}

func BenchmarkProofVerificationIssuer(b *testing.B) {
	bits, err := benchmark2.Bits(32, 64)
	require.NoError(b, err)
	curves := benchmark2.Curves(math.BN254, math.BLS12_381_BBS_GURVY, math2.BLS12_381_BBS_GURVY_FAST_RNG)
	outputs, err := benchmark2.NumOutputs(1, 2, 3)
	require.NoError(b, err)
	testCases := benchmark2.GenerateCases(bits, curves, nil, outputs, nil)

	for _, tc := range testCases {
		b.Run(tc.Name, func(b *testing.B) {
			env := newBenchmarkIssuerProofVerificationEnv(b, b.N, tc.BenchmarkCase)

			// Optional: Reset timer if you had expensive setup code above
			b.ResetTimer()

			i := 0
			for b.Loop() {
				// deserialize action
				action := &issue2.Action{}
				require.NoError(b, action.Deserialize(env.IssuerEnvs[i].actionRaw))

				// verify
				coms := make([]*math.G1, len(action.Outputs))
				for i := range len(action.Outputs) {
					coms[i] = action.Outputs[i].Data
				}
				require.NoError(b, issue2.NewVerifier(coms, env.IssuerEnvs[i].pp).Verify(action.GetProof()))
				i++
			}
		})
	}
}

type issuerEnv struct {
	pp           *v1.PublicParams
	outputValues []uint64
	outputOwners [][]byte
	actionRaw    []byte
}

func newIssuerEnv(pp *v1.PublicParams, numOutputs int) *issuerEnv {
	outputValues := make([]uint64, numOutputs)
	outputOwners := make([][]byte, numOutputs)
	for i := range outputValues {
		outputValues[i] = uint64(i*10 + 10)
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	return &issuerEnv{
		pp:           pp,
		outputValues: outputValues,
		outputOwners: outputOwners,
	}
}

func newIssuerProofVerificationEnv(tb testing.TB, pp *v1.PublicParams, numOutputs int) *issuerEnv {
	tb.Helper()
	outputValues := make([]uint64, numOutputs)
	outputOwners := make([][]byte, numOutputs)
	for i := range outputValues {
		outputValues[i] = uint64(i*10 + 10)
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	issuer := issue2.NewIssuer("ABC", &mock.SigningIdentity{}, pp)
	action, _, err := issuer.GenerateZKIssue(
		outputValues,
		outputOwners,
	)
	require.NoError(tb, err)
	actionRaw, err := action.Serialize()
	require.NoError(tb, err)

	return &issuerEnv{
		pp:        pp,
		actionRaw: actionRaw,
	}
}

type benchmarkIssuerEnv struct {
	IssuerEnvs []*issuerEnv
}

func newBenchmarkIssuerEnv(b *testing.B, n int, benchmarkCase *benchmark2.Case) *benchmarkIssuerEnv {
	b.Helper()
	envs := make([]*issuerEnv, n)
	pp := setup(b, benchmarkCase.Bits, benchmarkCase.CurveID)
	for i := range envs {
		envs[i] = newIssuerEnv(pp, benchmarkCase.NumOutputs)
	}
	return &benchmarkIssuerEnv{IssuerEnvs: envs}
}

func newBenchmarkIssuerProofVerificationEnv(b *testing.B, n int, benchmarkCase *benchmark2.Case) *benchmarkIssuerEnv {
	b.Helper()
	envs := make([]*issuerEnv, n)
	pp := setup(b, benchmarkCase.Bits, benchmarkCase.CurveID)
	for i := range envs {
		envs[i] = newIssuerProofVerificationEnv(b, pp, benchmarkCase.NumOutputs)
	}
	return &benchmarkIssuerEnv{IssuerEnvs: envs}
}

func setup(tb testing.TB, bits uint64, curveID math.CurveID) *v1.PublicParams {
	tb.Helper()
	pp, err := v1.Setup(bits, nil, curveID)
	require.NoError(tb, err)
	return pp
}

func prepareZKIssue(t *testing.T, bits uint64, curveID math.CurveID, numOutputs int) (*issue2.Prover, *issue2.Verifier) {
	t.Helper()
	pp, err := v1.Setup(bits, nil, curveID)
	assert.NoError(t, err)
	tw, tokens := prepareInputsForZKIssue(pp, numOutputs)
	prover, err := issue2.NewProver(tw, tokens, pp)
	assert.NoError(t, err)
	verifier := issue2.NewVerifier(tokens, pp)
	return prover, verifier
}

func prepareInputsForZKIssue(pp *v1.PublicParams, numOutputs int) ([]*token.Metadata, []*math.G1) {
	values := make([]uint64, numOutputs)
	for i := range numOutputs {
		values[i] = uint64(i*10 + 10)
	}
	curve := math.Curves[pp.Curve]
	rand, _ := curve.Rand()
	bf := make([]*math.Zr, len(values))
	for i := range values {
		bf[i] = math.Curves[pp.Curve].NewRandomZr(rand)
	}

	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = NewToken(curve.NewZrFromInt(int64(values[i])), bf[i], "ABC", pp.PedersenGenerators, curve)
	}
	return token.NewMetadata(pp.Curve, "ABC", values, bf), tokens
}
