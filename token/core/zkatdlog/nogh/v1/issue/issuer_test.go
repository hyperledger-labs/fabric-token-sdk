/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue_test

import (
	"fmt"
	"strconv"
	"testing"

	math "github.com/IBM/mathlib"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue/mock"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
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
	// Generate test cases programmatically instead of a static literal.
	bits := []uint64{32, 64}
	curves := []math.CurveID{math.BN254, math.BLS12_381_BBS_GURVY}
	outputs := []int{1, 2, 3}

	testCases := generateBenchmarkCases(bits, curves, outputs)

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			env := NewBenchmarkIssuerEnv(b, b.N, tc.benchmarkCase)

			// Optional: Reset timer if you had expensive setup code above
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				issuer := issue2.NewIssuer("ABC", &mock.SigningIdentity{}, env.IssuerEnvs[i].pp)
				action, _, err := issuer.GenerateZKIssue(
					env.IssuerEnvs[i].outputValues,
					env.IssuerEnvs[i].outputOwners,
				)
				require.NoError(b, err)
				_, err = action.Serialize()
				require.NoError(b, err)
			}
		})
	}
}

func BenchmarkIssuerProofVerification(b *testing.B) {
	// Generate test cases programmatically instead of a static literal.
	bits := []uint64{32, 64}
	curves := []math.CurveID{math.BN254, math.BLS12_381_BBS_GURVY}
	outputs := []int{1, 2, 3}

	testCases := generateBenchmarkCases(bits, curves, outputs)

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			env := NewBenchmarkIssuerProofVerificationEnv(b, b.N, tc.benchmarkCase)

			// Optional: Reset timer if you had expensive setup code above
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// deserialize action
				action := &issue2.Action{}
				require.NoError(b, action.Deserialize(env.IssuerEnvs[i].actionRaw))

				// verify
				coms := make([]*math.G1, len(action.Outputs))
				for i := range len(action.Outputs) {
					coms[i] = action.Outputs[i].Data
				}
				require.NoError(b, issue2.NewVerifier(coms, env.IssuerEnvs[i].pp).Verify(action.GetProof()))
			}
		})
	}
}

type BenchmarkCase struct {
	Bits       uint64
	CurveID    math.CurveID
	NumOutputs int
}

// generateBenchmarkCases returns all combinations of BenchmarkCase created
// from the provided slices of bits, curve IDs, number of inputs and outputs.
func generateBenchmarkCases(bits []uint64, curves []math.CurveID, outputs []int) []struct {
	name          string
	benchmarkCase *BenchmarkCase
} {
	var cases []struct {
		name          string
		benchmarkCase *BenchmarkCase
	}
	for _, b := range bits {
		for _, c := range curves {
			for _, ni := range outputs {
				name := fmt.Sprintf("Setup(bits %d, curve %s, #o %d)", b, math.CurveIDToString(c), ni)
				cases = append(cases, struct {
					name          string
					benchmarkCase *BenchmarkCase
				}{
					name: name,
					benchmarkCase: &BenchmarkCase{
						Bits:       b,
						CurveID:    c,
						NumOutputs: ni,
					},
				})
			}
		}
	}
	return cases
}

type IssuerEnv struct {
	pp           *v1.PublicParams
	outputValues []uint64
	outputOwners [][]byte
	actionRaw    []byte
}

func NewIssuerEnv(pp *v1.PublicParams, numOutputs int) *IssuerEnv {
	outputValues := make([]uint64, numOutputs)
	outputOwners := make([][]byte, numOutputs)
	for i := range outputValues {
		outputValues[i] = uint64(i*10 + 10)
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	return &IssuerEnv{
		pp:           pp,
		outputValues: outputValues,
		outputOwners: outputOwners,
	}
}

func NewIssuerProofVerificationEnv(tb testing.TB, pp *v1.PublicParams, numOutputs int) *IssuerEnv {
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

	return &IssuerEnv{
		pp:        pp,
		actionRaw: actionRaw,
	}
}

type BenchmarkIssuerEnv struct {
	IssuerEnvs []*IssuerEnv
}

func NewBenchmarkIssuerEnv(b *testing.B, n int, benchmarkCase *BenchmarkCase) *BenchmarkIssuerEnv {
	b.Helper()
	envs := make([]*IssuerEnv, n)
	pp := setup(b, benchmarkCase.Bits, benchmarkCase.CurveID)
	for i := range envs {
		envs[i] = NewIssuerEnv(pp, benchmarkCase.NumOutputs)
	}
	return &BenchmarkIssuerEnv{IssuerEnvs: envs}
}

func NewBenchmarkIssuerProofVerificationEnv(b *testing.B, n int, benchmarkCase *BenchmarkCase) *BenchmarkIssuerEnv {
	b.Helper()
	envs := make([]*IssuerEnv, n)
	pp := setup(b, benchmarkCase.Bits, benchmarkCase.CurveID)
	for i := range envs {
		envs[i] = NewIssuerProofVerificationEnv(b, pp, benchmarkCase.NumOutputs)
	}
	return &BenchmarkIssuerEnv{IssuerEnvs: envs}
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
