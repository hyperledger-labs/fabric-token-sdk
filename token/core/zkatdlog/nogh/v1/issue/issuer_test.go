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

// TestProverVerifier exercises a full prover -> verifier round-trip for
// a generated ZK-issue proof using the given curve and output count.
func TestProverVerifier(t *testing.T) {
	prover, verifier := prepareZKIssue(t, 32, math.BLS12_381_BBS_GURVY, 2)
	proof, err := prover.Prove()
	require.NoError(t, err)
	assert.NotNil(t, proof)
	err = verifier.Verify(proof)
	require.NoError(t, err)
}

// TestIssuer tests the high-level issuer API: generating a ZK issue
// action and verifying the resulting proof.
func TestIssuer(t *testing.T) {
	pp := setup(t, 32, math.BLS12_381_BBS_GURVY)
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

// BenchmarkIssuer measures the cost of creating ZK-issue actions under
// different parameter sets provided by the benchmark helper.
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
				e := env.IssuerEnvs[i%len(env.IssuerEnvs)]
				issuer := issue2.NewIssuer("ABC", &mock.SigningIdentity{}, e.pp)
				action, _, err := issuer.GenerateZKIssue(
					e.outputValues,
					e.outputOwners,
				)
				require.NoError(b, err)
				_, err = action.Serialize()
				require.NoError(b, err)
				i++
			}
		})
	}
}

// BenchmarkProofVerificationIssuer measures the cost of verifying a previously
// serialized ZK-issue action under different benchmark parameter sets.
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
				e := env.IssuerEnvs[i%len(env.IssuerEnvs)]
				// deserialize action
				action := &issue2.Action{}
				require.NoError(b, action.Deserialize(e.actionRaw))

				// verify
				coms := make([]*math.G1, len(action.Outputs))
				for i := range len(action.Outputs) {
					coms[i] = action.Outputs[i].Data
				}
				require.NoError(b, issue2.NewVerifier(coms, e.pp).Verify(action.GetProof()))
				i++
			}
		})
	}
}

// issuerEnv holds a prepared public parameters set together with a set of
// output values and owners used by the issuer benchmarks and tests.
type issuerEnv struct {
	pp           *v1.PublicParams
	outputValues []uint64
	outputOwners [][]byte
	actionRaw    []byte
}

// newIssuerEnv constructs an issuerEnv populated with deterministic values
// for the given number of outputs and the provided public parameters.
func newIssuerEnv(pp *v1.PublicParams, numOutputs int) *issuerEnv {
	outputValues := make([]uint64, numOutputs)
	outputOwners := make([][]byte, numOutputs)
	for i := range outputValues {
		outputValues[i] = uint64(i*10 + 10) // #nosec G115
		outputOwners[i] = []byte("alice_" + strconv.Itoa(i))
	}

	return &issuerEnv{
		pp:           pp,
		outputValues: outputValues,
		outputOwners: outputOwners,
	}
}

// newIssuerProofVerificationEnv prepares an issuerEnv containing a serialized
// action that can be used to measure verification cost.
func newIssuerProofVerificationEnv(tb testing.TB, pp *v1.PublicParams, numOutputs int) *issuerEnv {
	tb.Helper()
	outputValues := make([]uint64, numOutputs)
	outputOwners := make([][]byte, numOutputs)
	for i := range outputValues {
		outputValues[i] = uint64(i*10 + 10) // #nosec G115
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

// benchmarkIssuerEnv groups a slice of prepared issuerEnv used by the
// benchmark harness.
type benchmarkIssuerEnv struct {
	IssuerEnvs []*issuerEnv
}

// newBenchmarkIssuerEnv creates n prepared issuer environments using the
// provided benchmark case parameters.
func newBenchmarkIssuerEnv(b *testing.B, n int, benchmarkCase *benchmark2.Case) *benchmarkIssuerEnv {
	b.Helper()
	envs := make([]*issuerEnv, n)
	pp := setup(b, benchmarkCase.Bits, benchmarkCase.CurveID)
	for i := range envs {
		envs[i] = newIssuerEnv(pp, benchmarkCase.NumOutputs)
	}

	return &benchmarkIssuerEnv{IssuerEnvs: envs}
}

// newBenchmarkIssuerProofVerificationEnv creates n prepared issuer
// environments where each entry contains a serialized action to be
// verified in benchmarks.
func newBenchmarkIssuerProofVerificationEnv(b *testing.B, n int, benchmarkCase *benchmark2.Case) *benchmarkIssuerEnv {
	b.Helper()
	envs := make([]*issuerEnv, n)
	pp := setup(b, benchmarkCase.Bits, benchmarkCase.CurveID)
	for i := range envs {
		envs[i] = newIssuerProofVerificationEnv(b, pp, benchmarkCase.NumOutputs)
	}

	return &benchmarkIssuerEnv{IssuerEnvs: envs}
}

// setup initializes public parameters for tests and benchmarks using the
// provided bit-size and curve identifier.
func setup(tb testing.TB, bits uint64, curveID math.CurveID) *v1.PublicParams {
	tb.Helper()
	pp, err := v1.Setup(bits, nil, curveID)
	require.NoError(tb, err)

	return pp
}

// prepareZKIssue prepares a prover and verifier pair along with token inputs
// for a ZK-issue proof using the provided bits, curve and number of outputs.
func prepareZKIssue(t *testing.T, bits uint64, curveID math.CurveID, numOutputs int) (*issue2.Prover, *issue2.Verifier) {
	t.Helper()
	pp, err := v1.Setup(bits, nil, curveID)
	require.NoError(t, err)
	tw, tokens := prepareInputsForZKIssue(pp, numOutputs)
	prover, err := issue2.NewProver(tw, tokens, pp)
	require.NoError(t, err)
	verifier := issue2.NewVerifier(tokens, pp)

	return prover, verifier
}

// prepareInputsForZKIssue creates deterministic token metadata and token
// commitments that serve as inputs to the prover/verifier in tests.
func prepareInputsForZKIssue(pp *v1.PublicParams, numOutputs int) ([]*token.Metadata, []*math.G1) {
	values := make([]uint64, numOutputs)
	for i := range numOutputs {
		values[i] = uint64(i*10 + 10) //nolint:gosec
	}
	curve := math.Curves[pp.Curve]
	rand, _ := curve.Rand()
	bf := make([]*math.Zr, len(values))
	for i := range values {
		bf[i] = math.Curves[pp.Curve].NewRandomZr(rand)
	}

	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = NewToken(curve.NewZrFromInt(int64(values[i])), bf[i], "ABC", pp.PedersenGenerators, curve) // #nosec G115
	}

	return token.NewMetadata(pp.Curve, "ABC", values, bf), tokens
}

func TestIssuerSignTokenActions(t *testing.T) {
	pp := setup(t, 32, math.BLS12_381_BBS_GURVY)
	signer := &mock.SigningIdentity{}
	issuer := issue2.NewIssuer("ABC", signer, pp)

	raw := []byte("hello")
	signer.SignReturns([]byte("signature"), nil)

	sig, err := issuer.SignTokenActions(raw)
	assert.NoError(t, err)
	assert.Equal(t, []byte("signature"), sig)
	assert.Equal(t, 1, signer.SignCallCount())
	assert.Equal(t, raw, signer.SignArgsForCall(0))

	// Signer nil
	issuer.Signer = nil
	_, err = issuer.SignTokenActions(raw)
	assert.EqualError(t, err, "failed to sign Token Actions: please initialize signer")
}

func TestIssuerGenerateZKIssueErrors(t *testing.T) {
	pp := setup(t, 32, math.BLS12_381_BBS_GURVY)
	issuer := issue2.NewIssuer("ABC", &mock.SigningIdentity{}, pp)

	// PublicParams nil
	issuer.PublicParams = nil
	_, _, err := issuer.GenerateZKIssue([]uint64{10}, [][]byte{[]byte("alice")})
	assert.EqualError(t, err, "failed to generate ZK Issue: nil public parameters")
	issuer.PublicParams = pp

	// Inadmissible curve
	oldCurve := issuer.PublicParams.Curve
	issuer.PublicParams.Curve = math.CurveID(len(math.Curves) + 1)
	_, _, err = issuer.GenerateZKIssue([]uint64{10}, [][]byte{[]byte("alice")})
	assert.EqualError(t, err, "failed to generate ZK Issue: please initialize public parameters with an admissible curve")
	issuer.PublicParams.Curve = oldCurve

	// Signer nil
	issuer.Signer = nil
	_, _, err = issuer.GenerateZKIssue([]uint64{10}, [][]byte{[]byte("alice")})
	assert.EqualError(t, err, "failed to generate ZK Issue: please initialize signer")
}
