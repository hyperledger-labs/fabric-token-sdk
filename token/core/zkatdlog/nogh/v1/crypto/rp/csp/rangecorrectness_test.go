/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSPRangeCorrectnessProveVerify verifies batch range proof generation and verification.
func TestCSPRangeCorrectnessProveVerify(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)
	numCommitments := 3

	// Generate generators
	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	// Generate commitments and values
	commitments := make([]*math.G1, numCommitments)
	values := make([]uint64, numCommitments)
	blindingFactors := make([]*math.Zr, numCommitments)

	for i := range numCommitments {
		values[i] = uint64(i + 1)
		blindingFactors[i] = curve.NewRandomZr(rand)
		v := curve.NewZrFromUint64(values[i])
		commitments[i] = curve.MultiScalarMul(pedersenParams, []*math.Zr{v, blindingFactors[i]})
	}

	// Prove
	prover := NewCSPRangeCorrectnessProver(
		commitments,
		values,
		blindingFactors,
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)
	require.NotNil(t, rc)
	require.Len(t, rc.Proofs, numCommitments)

	// Verify
	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = commitments

	err = verifier.Verify(rc)
	require.NoError(t, err)
}

// TestCSPRangeCorrectnessSingleCommitment verifies batch proof with single commitment.
func TestCSPRangeCorrectnessSingleCommitment(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	value := uint64(42)
	blindingFactor := curve.NewRandomZr(rand)
	v := curve.NewZrFromUint64(value)
	commitment := curve.MultiScalarMul(pedersenParams, []*math.Zr{v, blindingFactor})

	prover := NewCSPRangeCorrectnessProver(
		[]*math.G1{commitment},
		[]uint64{value},
		[]*math.Zr{blindingFactor},
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)
	require.Len(t, rc.Proofs, 1)

	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = []*math.G1{commitment}

	err = verifier.Verify(rc)
	require.NoError(t, err)
}

// TestCSPRangeCorrectnessEmptyCommitments verifies behavior with empty commitment set.
func TestCSPRangeCorrectnessEmptyCommitments(t *testing.T) {
	curve := math.Curves[math.BN254]
	n := uint64(8)

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	prover := NewCSPRangeCorrectnessProver(
		[]*math.G1{},
		[]uint64{},
		[]*math.Zr{},
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)
	require.Len(t, rc.Proofs, 0)

	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = []*math.G1{}

	err = verifier.Verify(rc)
	require.NoError(t, err)
}

// TestCSPRangeCorrectnessMismatchedProofCount verifies error when proof count doesn't match.
func TestCSPRangeCorrectnessMismatchedProofCount(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	// Create 2 commitments
	commitments := make([]*math.G1, 2)
	for i := range 2 {
		v := curve.NewZrFromUint64(uint64(i + 1))
		r := curve.NewRandomZr(rand)
		commitments[i] = curve.MultiScalarMul(pedersenParams, []*math.Zr{v, r})
	}

	// Create proof with only 1 proof
	rc := &CSPRangeCorrectness{
		Proofs: []*CspRangeProof{{}},
	}

	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = commitments

	err = verifier.Verify(rc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid range proof")
}

// TestCSPRangeCorrectnessNilProof verifies error when a proof is nil.
func TestCSPRangeCorrectnessNilProof(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	commitment := curve.GenG1.Mul(curve.NewRandomZr(rand))

	rc := &CSPRangeCorrectness{
		Proofs: []*CspRangeProof{nil},
	}

	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = []*math.G1{commitment}

	err = verifier.Verify(rc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil proof")
}

// TestCSPRangeCorrectnessSerializationRoundTrip verifies serialization and deserialization.
func TestCSPRangeCorrectnessSerializationRoundTrip(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)
	numCommitments := 2

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	commitments := make([]*math.G1, numCommitments)
	values := make([]uint64, numCommitments)
	blindingFactors := make([]*math.Zr, numCommitments)

	for i := range numCommitments {
		values[i] = uint64(i + 1)
		blindingFactors[i] = curve.NewRandomZr(rand)
		v := curve.NewZrFromUint64(values[i])
		commitments[i] = curve.MultiScalarMul(pedersenParams, []*math.Zr{v, blindingFactors[i]})
	}

	prover := NewCSPRangeCorrectnessProver(
		commitments,
		values,
		blindingFactors,
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)

	// Serialize
	serialized, err := rc.Serialize()
	require.NoError(t, err)
	require.NotEmpty(t, serialized)

	// Deserialize
	rc2 := &CSPRangeCorrectness{}
	err = rc2.Deserialize(serialized)
	require.NoError(t, err)
	require.Len(t, rc2.Proofs, numCommitments)

	// Verify deserialized proof
	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = commitments

	err = verifier.Verify(rc2)
	require.NoError(t, err)
}

// TestCSPRangeCorrectnessValidate verifies the Validate method.
func TestCSPRangeCorrectnessValidate(t *testing.T) {
	testCases := []struct {
		name      string
		rc        *CSPRangeCorrectness
		curveID   math.CurveID
		expectErr bool
	}{
		{
			name: "valid_empty",
			rc: &CSPRangeCorrectness{
				Proofs: []*CspRangeProof{},
			},
			curveID:   math.BN254,
			expectErr: false,
		},
		{
			name: "valid_single",
			rc: &CSPRangeCorrectness{
				Proofs: []*CspRangeProof{{}},
			},
			curveID:   math.BN254,
			expectErr: false,
		},
		{
			name: "nil_proof",
			rc: &CSPRangeCorrectness{
				Proofs: []*CspRangeProof{nil},
			},
			curveID:   math.BN254,
			expectErr: true,
		},
		{
			name: "mixed_nil",
			rc: &CSPRangeCorrectness{
				Proofs: []*CspRangeProof{{}, nil, {}},
			},
			curveID:   math.BN254,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rc.Validate(tc.curveID)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCSPRangeCorrectnessLargeSet verifies batch proof with many commitments.
func TestCSPRangeCorrectnessLargeSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large test in short mode")
	}

	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)
	numCommitments := 10

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	commitments := make([]*math.G1, numCommitments)
	values := make([]uint64, numCommitments)
	blindingFactors := make([]*math.Zr, numCommitments)

	for i := range numCommitments {
		values[i] = uint64(i * 10)
		blindingFactors[i] = curve.NewRandomZr(rand)
		v := curve.NewZrFromUint64(values[i])
		commitments[i] = curve.MultiScalarMul(pedersenParams, []*math.Zr{v, blindingFactors[i]})
	}

	prover := NewCSPRangeCorrectnessProver(
		commitments,
		values,
		blindingFactors,
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)
	require.Len(t, rc.Proofs, numCommitments)

	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = commitments

	err = verifier.Verify(rc)
	require.NoError(t, err)
}

// TestCSPRangeCorrectnessBoundaryValues verifies proofs for boundary values.
func TestCSPRangeCorrectnessBoundaryValues(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8) // Range [0, 255]

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	boundaryValues := []uint64{0, 1, 127, 128, 254, 255}
	commitments := make([]*math.G1, len(boundaryValues))
	blindingFactors := make([]*math.Zr, len(boundaryValues))

	for i, val := range boundaryValues {
		blindingFactors[i] = curve.NewRandomZr(rand)
		v := curve.NewZrFromUint64(val)
		commitments[i] = curve.MultiScalarMul(pedersenParams, []*math.Zr{v, blindingFactors[i]})
	}

	prover := NewCSPRangeCorrectnessProver(
		commitments,
		boundaryValues,
		blindingFactors,
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)

	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = commitments

	err = verifier.Verify(rc)
	require.NoError(t, err)
}

// TestCSPRangeCorrectnessWrongCommitment verifies detection of wrong commitment.
func TestCSPRangeCorrectnessWrongCommitment(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)

	pedersenParams := []*math.G1{
		curve.HashToG1([]byte("ped-0")),
		curve.HashToG1([]byte("ped-1")),
	}
	leftGens := make([]*math.G1, n+1)
	rightGens := make([]*math.G1, n+1)
	for i := uint64(0); i <= n; i++ {
		leftGens[i] = curve.HashToG1([]byte{byte(i), 0})
		rightGens[i] = curve.HashToG1([]byte{byte(i), 1})
	}

	value := uint64(42)
	blindingFactor := curve.NewRandomZr(rand)
	v := curve.NewZrFromUint64(value)
	commitment := curve.MultiScalarMul(pedersenParams, []*math.Zr{v, blindingFactor})

	prover := NewCSPRangeCorrectnessProver(
		[]*math.G1{commitment},
		[]uint64{value},
		[]*math.Zr{blindingFactor},
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)

	// Use wrong commitment for verification
	wrongCommitment := curve.GenG1.Mul(curve.NewRandomZr(rand))

	verifier := NewCSPRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		n,
		curve,
	)
	verifier.Commitments = []*math.G1{wrongCommitment}

	err = verifier.Verify(rc)
	require.Error(t, err)
}

// TestCSPRangeCorrectnessDeserializeInvalid verifies error handling for invalid serialization.
func TestCSPRangeCorrectnessDeserializeInvalid(t *testing.T) {
	rc := &CSPRangeCorrectness{}

	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "empty",
			data: []byte{},
		},
		{
			name: "invalid_asn1",
			data: []byte{0xFF, 0xFF, 0xFF},
		},
		{
			name: "truncated",
			data: []byte{0x30, 0x10, 0x00},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := rc.Deserialize(tc.data)
			assert.Error(t, err)
		})
	}
}
