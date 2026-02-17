/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/stretchr/testify/require"
)

func TestRangeCorrectness(t *testing.T) {
	curve := math.Curves[math.BN254]
	bitLength := uint64(32)
	rounds := uint64(5) // for 32 bits

	rand, err := curve.Rand()
	require.NoError(t, err)

	pedersenParams := []*math.G1{curve.GenG1, curve.GenG1.Mul(curve.NewRandomZr(rand))}
	leftGens := make([]*math.G1, bitLength)
	rightGens := make([]*math.G1, bitLength)
	for i := range bitLength {
		leftGens[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
		rightGens[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}
	P := curve.GenG1.Mul(curve.NewRandomZr(rand))
	Q := curve.GenG1.Mul(curve.NewRandomZr(rand))

	values := []uint64{10, 20}
	blindingFactors := []*math.Zr{curve.NewRandomZr(rand), curve.NewRandomZr(rand)}
	commitments := make([]*math.G1, len(values))
	for i := range values {
		commitments[i] = pedersenParams[0].Mul(curve.NewZrFromUint64(values[i]))
		commitments[i].Add(pedersenParams[1].Mul(blindingFactors[i]))
	}

	prover := NewRangeCorrectnessProver(
		commitments,
		values,
		blindingFactors,
		pedersenParams,
		leftGens,
		rightGens,
		P,
		Q,
		bitLength,
		rounds,
		curve,
	)

	rc, err := prover.Prove()
	require.NoError(t, err)
	require.NotNil(t, rc)
	require.Len(t, rc.Proofs, 2)

	// Test Validate
	err = rc.Validate(math.BN254)
	require.NoError(t, err)

	// Test Serialize/Deserialize
	raw, err := rc.Serialize()
	require.NoError(t, err)
	rc2 := &RangeCorrectness{}
	err = rc2.Deserialize(raw)
	require.NoError(t, err)
	require.Len(t, rc2.Proofs, 2)

	// Test Verify
	verifier := NewRangeCorrectnessVerifier(
		pedersenParams,
		leftGens,
		rightGens,
		P,
		Q,
		bitLength,
		rounds,
		curve,
	)
	// We need to manually set commitments because NewRangeCorrectnessVerifier doesn't take them
	verifier.Commitments = commitments

	err = verifier.Verify(rc)
	require.NoError(t, err)

	// Test Verify Error (wrong number of commitments)
	verifier.Commitments = commitments[:1]
	err = verifier.Verify(rc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid range proof")

	// Test Validate Error (nil proof)
	rcNil := &RangeCorrectness{Proofs: []*RangeProof{nil}}
	err = rcNil.Validate(math.BN254)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid range proof: nil proof at index 0")

	// Test Validate Error (inner proof validation failure)
	rcInvalid := &RangeCorrectness{Proofs: []*RangeProof{{Data: &RangeProofData{}}}}
	err = rcInvalid.Validate(math.BN254)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid range proof at index 0")

	// Test Deserialize Error
	rc3 := &RangeCorrectness{}
	err = rc3.Deserialize([]byte("invalid"))
	require.Error(t, err)

	// Test Verify Error (nil proof)
	verifier.Commitments = []*math.G1{commitments[0]}
	err = verifier.Verify(rcNil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid range proof: nil proof at index 0")
}
