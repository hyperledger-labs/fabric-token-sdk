/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"fmt"
	"testing"

	mathlib "github.com/IBM/mathlib"
)

// TestAllCurvesSupported verifies that all critical tests run successfully
// with both BLS12_381_BBS_GURVY and BN254 curves.
// Given a set of supported curves,
// When critical protocol operations (transcript, CSP, range proof, validation) are tested,
// Then all operations should succeed on each curve.
func TestAllCurvesSupported(t *testing.T) {
	curves := []mathlib.CurveID{
		mathlib.BLS12_381_BBS_GURVY,
		mathlib.BN254,
	}

	for _, curveID := range curves {
		t.Run(fmt.Sprintf("%d", curveID), func(t *testing.T) {
			curve := mathlib.Curves[curveID]

			// Run a subset of critical tests to verify curve support
			t.Run("Transcript", func(t *testing.T) {
				testTranscriptBasicOperations(t, curve)
			})

			t.Run("CSP", func(t *testing.T) {
				testCSPProveVerify(t, curve)
			})

			t.Run("RangeProof", func(t *testing.T) {
				testRangeProofBasic(t, curve)
			})

			t.Run("Validation", func(t *testing.T) {
				testValidationFunctions(t, curve)
			})
		})
	}
}

func testTranscriptBasicOperations(t *testing.T, curve *mathlib.Curve) {
	t.Helper()
	tr := Transcript{Curve: curve}
	tr.InitHasher()

	// Test absorb and squeeze
	tr.Absorb([]byte("test data"))
	challenge, err := tr.Squeeze()
	if err != nil {
		t.Fatalf("Failed to squeeze challenge: %v", err)
	}
	if challenge == nil {
		t.Fatal("Challenge should not be nil")
	}
}

func testCSPProveVerify(t *testing.T, curve *mathlib.Curve) {
	t.Helper()
	rand, err := curve.Rand()
	if err != nil {
		t.Fatalf("Failed to create random generator: %v", err)
	}

	rounds := uint64(2)
	size := 1 << rounds

	// Create generators and witness
	gens := make([]*mathlib.G1, size)
	wit := make([]*mathlib.Zr, size)
	for i := range size {
		gens[i] = curve.GenG1
		wit[i] = curve.NewRandomZr(rand)
	}

	// Create linear form
	lf := make([]*mathlib.Zr, size)
	for i := range size {
		lf[i] = curve.NewRandomZr(rand)
	}

	// Compute commitment and value
	comm := curve.MultiScalarMul(gens, wit)
	val := curve.NewZrFromInt(0)
	for i := range size {
		val = curve.ModAdd(val, curve.ModMul(lf[i], wit[i], curve.GroupOrder), curve.GroupOrder)
	}

	// Create prover and generate proof
	prover := &cspProver{
		Commitment:     comm,
		Generators:     gens,
		LinearForm:     lf,
		Value:          val,
		NumberOfRounds: rounds,
		Curve:          curve,
		witness:        wit,
	}

	proof, err := prover.Prove()
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Verify proof
	verifier := &cspVerifier{
		Commitment:     comm,
		Generators:     gens,
		LinearForm:     lf,
		Value:          val,
		NumberOfRounds: rounds,
		Curve:          curve,
	}

	err = verifier.Verify(proof)
	if err != nil {
		t.Fatalf("Failed to verify proof: %v", err)
	}
}

func testRangeProofBasic(t *testing.T, curve *mathlib.Curve) {
	t.Helper()
	rand, err := curve.Rand()
	if err != nil {
		t.Fatalf("Failed to create random generator: %v", err)
	}

	n := uint64(8)
	v := curve.NewZrFromUint64(42) // Value in range [0, 2^8-1]
	r := curve.NewRandomZr(rand)

	// Create generators
	vGens := []*mathlib.G1{curve.GenG1, curve.GenG1}
	aGens := make([]*mathlib.G1, n+1)
	bGens := make([]*mathlib.G1, n+1)
	for i := range aGens {
		aGens[i] = curve.GenG1
		bGens[i] = curve.GenG1
	}

	// Create commitment
	vComm := curve.MultiScalarMul(vGens, []*mathlib.Zr{v, r})

	// Create prover and generate proof
	prover := NewCspRangeProver(vComm, v, r, vGens, aGens, bGens, n, curve)
	proof, err := prover.Prove()
	if err != nil {
		t.Fatalf("Failed to generate range proof: %v", err)
	}

	// Verify proof
	verifier := newCspRangeVerifier(vGens, aGens, bGens, vComm, n, curve)
	err = verifier.Verify(proof)
	if err != nil {
		t.Fatalf("Failed to verify range proof: %v", err)
	}
}

func testValidationFunctions(t *testing.T, curve *mathlib.Curve) {
	t.Helper()
	// Test curve validation
	err := validateCurve(curve)
	if err != nil {
		t.Fatalf("Valid curve failed validation: %v", err)
	}

	// Test G1 slice validation
	g1Slice := []*mathlib.G1{curve.GenG1, curve.GenG1}
	err = validateG1Slice("test", g1Slice, curve, 2)
	if err != nil {
		t.Fatalf("Valid G1 slice failed validation: %v", err)
	}

	// Test Zr slice validation
	zrSlice := []*mathlib.Zr{curve.NewZrFromInt(1), curve.NewZrFromInt(2)}
	err = validateZrSlice("test", zrSlice, curve, 2)
	if err != nil {
		t.Fatalf("Valid Zr slice failed validation: %v", err)
	}
}
