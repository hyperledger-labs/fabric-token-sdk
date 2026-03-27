/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"fmt"
	"testing"

	mathlib "github.com/IBM/mathlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCurve verifies that the validateCurve function correctly validates
// elliptic curve instances used in CSP proofs. This is a foundational validation
// that ensures all cryptographic operations have a valid curve context.
//
// Test coverage:
// - Validates that non-nil curve instances from the mathlib are accepted
// - Ensures nil curve pointers are rejected with appropriate error messages
// - Runs for both BN254 and BLS12_381_BBS curves to ensure cross-curve compatibility
func TestValidateCurve(t *testing.T) {
	curves := []mathlib.CurveID{
		mathlib.BLS12_381_BBS,
		mathlib.BN254,
	}

	for _, curveID := range curves {
		t.Run(fmt.Sprintf("%d", curveID), func(t *testing.T) {
			curve := mathlib.Curves[curveID]

			t.Run("ValidCurve", func(t *testing.T) {
				// Verify that a valid curve instance passes validation
				err := validateCurve(curve)
				require.NoError(t, err)
			})

			t.Run("NilCurve", func(t *testing.T) {
				// Verify that nil curve is rejected to prevent nil pointer dereferences
				err := validateCurve(nil)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "curve cannot be nil")
			})
		})
	}
}

// TestValidateG1Slice verifies that G1 point slices (elliptic curve group elements)
// are properly validated before use in CSP proofs. G1 elements represent commitments,
// generators, and proof components in the cryptographic protocol.
//
// Test coverage:
// - Validates that slices with correct length and valid G1 points pass validation
// - Ensures nil slices are rejected to prevent nil pointer dereferences
// - Verifies that slices with incorrect length are rejected (critical for protocol security)
// - Confirms that slices containing nil elements are detected and rejected
// - Tests both BN254 and BLS12_381_BBS curves to ensure cross-curve compatibility
//
// Note: The validation allows infinity points (identity elements) as they are valid
// in CSP proofs for zero witnesses and identity generators.
func TestValidateG1Slice(t *testing.T) {
	curves := []mathlib.CurveID{
		mathlib.BLS12_381_BBS,
		mathlib.BN254,
	}

	for _, curveID := range curves {
		t.Run(fmt.Sprintf("%d", curveID), func(t *testing.T) {
			curve := mathlib.Curves[curveID]

			t.Run("ValidSlice", func(t *testing.T) {
				// Verify that a slice of valid G1 points with correct length passes validation
				elements := []*mathlib.G1{curve.GenG1, curve.GenG1}
				err := validateG1Slice("test", elements, curve, 2)
				require.NoError(t, err)
			})

			t.Run("NilSlice", func(t *testing.T) {
				// Verify that nil slice is rejected to prevent nil pointer dereferences
				err := validateG1Slice("test", nil, curve, 2)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "test cannot be nil")
			})

			t.Run("WrongLength", func(t *testing.T) {
				// Verify that slices with incorrect length are rejected (security critical)
				elements := []*mathlib.G1{curve.GenG1}
				err := validateG1Slice("test", elements, curve, 2)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid length")
			})

			t.Run("NilElement", func(t *testing.T) {
				// Verify that slices containing nil elements are detected and rejected
				elements := []*mathlib.G1{curve.GenG1, nil}
				err := validateG1Slice("test", elements, curve, 2)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "element cannot be nil")
				assert.Contains(t, err.Error(), "test[1]")
			})
		})
	}
}

// TestValidateZrSlice verifies that Zr scalar slices (field elements) are properly
// validated before use in CSP proofs. Zr elements represent witnesses, challenges,
// responses, and other scalar values in the cryptographic protocol.
//
// Test coverage:
// - Validates that slices with correct length and valid Zr scalars pass validation
// - Ensures nil slices are rejected to prevent nil pointer dereferences
// - Verifies that slices with incorrect length are rejected (critical for protocol security)
// - Confirms that slices containing nil elements are detected and rejected
// - Tests both BN254 and BLS12_381_BBS curves to ensure cross-curve compatibility
//
// Note: Zr elements must belong to the correct curve's scalar field. The validation
// uses the math package's CheckZrElements utility to ensure curve consistency.
func TestValidateZrSlice(t *testing.T) {
	curves := []mathlib.CurveID{
		mathlib.BLS12_381_BBS,
		mathlib.BN254,
	}

	for _, curveID := range curves {
		t.Run(fmt.Sprintf("%d", curveID), func(t *testing.T) {
			curve := mathlib.Curves[curveID]

			t.Run("ValidSlice", func(t *testing.T) {
				// Verify that a slice of valid Zr scalars with correct length passes validation
				elements := []*mathlib.Zr{curve.NewZrFromInt(1), curve.NewZrFromInt(2)}
				err := validateZrSlice("test", elements, curve, 2)
				require.NoError(t, err)
			})

			t.Run("NilSlice", func(t *testing.T) {
				// Verify that nil slice is rejected to prevent nil pointer dereferences
				err := validateZrSlice("test", nil, curve, 2)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "test cannot be nil")
			})

			t.Run("WrongLength", func(t *testing.T) {
				// Verify that slices with incorrect length are rejected (security critical)
				elements := []*mathlib.Zr{curve.NewZrFromInt(1)}
				err := validateZrSlice("test", elements, curve, 2)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid length")
			})

			t.Run("NilElement", func(t *testing.T) {
				// Verify that slices containing nil elements are detected and rejected
				elements := []*mathlib.Zr{curve.NewZrFromInt(1), nil}
				err := validateZrSlice("test", elements, curve, 2)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "validation failed")
			})
		})
	}
}

// TestValidateCSPProverInputs verifies that all inputs to the CSP prover are properly
// validated before proof generation. The CSP (Compressed Sigma Protocol) prover requires
// specific structural constraints on its inputs to ensure correct and secure proof generation.
//
// Test coverage:
// - Validates that a properly constructed prover with all required fields passes validation
// - Ensures nil curve is rejected (required for all cryptographic operations)
// - Verifies that nil commitment is rejected (the commitment is the public statement being proven)
// - Confirms that generators with incorrect length are rejected (must match 2^rounds)
// - Tests both BN254 and BLS12_381_BBS curves to ensure cross-curve compatibility
//
// The prover structure includes:
// - Commitment: The public G1 point being proven (C = <generators, witness>)
// - Generators: G1 points used in the inner product (length must be 2^rounds)
// - LinearForm: Zr scalars defining the linear form being proven (length must be 2^rounds)
// - Value: The claimed evaluation of the linear form
// - NumberOfRounds: Determines the proof size (log2 of vector length)
// - witness: Private Zr scalars satisfying the commitment equation
func TestValidateCSPProverInputs(t *testing.T) {
	curves := []mathlib.CurveID{
		mathlib.BLS12_381_BBS,
		mathlib.BN254,
	}

	for _, curveID := range curves {
		t.Run(fmt.Sprintf("%d", curveID), func(t *testing.T) {
			curve := mathlib.Curves[curveID]
			rand, err := curve.Rand()
			require.NoError(t, err)

			// Create valid prover with 2^2 = 4 elements
			rounds := uint64(2)
			size := 1 << rounds
			gens := make([]*mathlib.G1, size)
			lf := make([]*mathlib.Zr, size)
			wit := make([]*mathlib.Zr, size)
			for i := range size {
				gens[i] = curve.GenG1
				lf[i] = curve.NewRandomZr(rand)
				wit[i] = curve.NewRandomZr(rand)
			}

			p := &cspProver{
				Commitment:     curve.GenG1,
				Generators:     gens,
				LinearForm:     lf,
				Value:          curve.NewRandomZr(rand),
				NumberOfRounds: rounds,
				Curve:          curve,
				witness:        wit,
			}

			t.Run("ValidInputs", func(t *testing.T) {
				// Verify that a properly constructed prover passes all validation checks
				err := validateCSPProverInputs(p)
				require.NoError(t, err)
			})

			t.Run("NilCurve", func(t *testing.T) {
				// Verify that nil curve is rejected (required for all crypto operations)
				pCopy := *p
				pCopy.Curve = nil
				err := validateCSPProverInputs(&pCopy)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid curve")
			})

			t.Run("NilCommitment", func(t *testing.T) {
				// Verify that nil commitment is rejected (the public statement being proven)
				pCopy := *p
				pCopy.Commitment = nil
				err := validateCSPProverInputs(&pCopy)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "commitment cannot be nil")
			})

			t.Run("WrongGeneratorsLength", func(t *testing.T) {
				// Verify that generators with incorrect length are rejected (must be 2^rounds)
				pCopy := *p
				pCopy.Generators = gens[:size-1]
				err := validateCSPProverInputs(&pCopy)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid length")
			})
		})
	}
}

// TestValidateRangeProverInputs verifies that all inputs to the range prover are properly
// validated before proof generation. The range prover proves that a committed value lies
// within a specific range [0, 2^n) without revealing the value itself.
//
// Test coverage:
// - Validates that a properly constructed range prover with all required fields passes validation
// - Ensures nil value is rejected (the secret value being range-proven)
// - Verifies that zero bits is rejected (must prove at least 1 bit)
// - Confirms that more than 64 bits is rejected (implementation limit)
// - Tests both BN254 and BLS12_381_BBS curves to ensure cross-curve compatibility
//
// The range prover structure includes:
// - VCommitment: Public commitment to the value being range-proven (C = v*G + r*H)
// - v: The secret value being proven to be in range [0, 2^n)
// - r: The randomness used in the commitment (for hiding)
// - VGenerators: G1 points [G, H] used for the Pedersen commitment
// - AGenerators: G1 points for the bit decomposition (length n+1)
// - BGenerators: G1 points for the bit decomposition (length n+1)
// - NumberOfBits: The bit length n defining the range [0, 2^n)
func TestValidateRangeProverInputs(t *testing.T) {
	curves := []mathlib.CurveID{
		mathlib.BLS12_381_BBS,
		mathlib.BN254,
	}

	for _, curveID := range curves {
		t.Run(fmt.Sprintf("%d", curveID), func(t *testing.T) {
			curve := mathlib.Curves[curveID]
			rand, err := curve.Rand()
			require.NoError(t, err)

			// Create valid range prover for 8-bit range [0, 256)
			n := uint64(8)
			vGens := []*mathlib.G1{curve.GenG1, curve.GenG1}
			aGens := make([]*mathlib.G1, n+1)
			bGens := make([]*mathlib.G1, n+1)
			for i := range aGens {
				aGens[i] = curve.GenG1
				bGens[i] = curve.GenG1
			}

			p := &cspRangeProver{
				VCommitment:  curve.GenG1,
				v:            curve.NewRandomZr(rand),
				r:            curve.NewRandomZr(rand),
				VGenerators:  vGens,
				AGenerators:  aGens,
				BGenerators:  bGens,
				NumberOfBits: n,
				Curve:        curve,
			}

			t.Run("ValidInputs", func(t *testing.T) {
				// Verify that a properly constructed range prover passes all validation checks
				err := validateRangeProverInputs(p)
				require.NoError(t, err)
			})

			t.Run("NilValue", func(t *testing.T) {
				// Verify that nil value is rejected (the secret being range-proven)
				pCopy := *p
				pCopy.v = nil
				err := validateRangeProverInputs(&pCopy)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "value cannot be nil")
			})

			t.Run("ZeroBits", func(t *testing.T) {
				// Verify that zero bits is rejected (must prove at least 1 bit)
				pCopy := *p
				pCopy.NumberOfBits = 0
				err := validateRangeProverInputs(&pCopy)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid number of bits")
				assert.Contains(t, err.Error(), "must be greater than 0")
			})

			t.Run("TooManyBits", func(t *testing.T) {
				// Verify that more than 64 bits is rejected (implementation limit)
				pCopy := *p
				pCopy.NumberOfBits = 65
				err := validateRangeProverInputs(&pCopy)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid number of bits")
				assert.Contains(t, err.Error(), "cannot exceed 64")
			})
		})
	}
}

// TestValidateRangeProof verifies that range proofs are properly validated before
// verification. A range proof demonstrates that a committed value lies within [0, 2^n)
// without revealing the value. The proof structure combines a CSP proof with additional
// components for the range constraint.
//
// Test coverage:
// - Validates that a properly constructed range proof with all required fields passes validation
// - Ensures nil proof is rejected to prevent nil pointer dereferences
// - Verifies that nil pComm (polynomial commitment) is rejected
// - Tests both BN254 and BLS12_381_BBS curves to ensure cross-curve compatibility
//
// The range proof structure includes:
// - pComm: Commitment to the polynomial representing the bit decomposition
// - pokV: Proof of knowledge for the value commitment (includes A point and Z responses)
// - u: Challenge scalar used in the Fiat-Shamir transformation
// - sComm: Commitment to the polynomial evaluation
// - sEval: The evaluation of the polynomial at the challenge point
// - cspProof: The underlying CSP proof demonstrating the inner product relation
//
// Note: The CSP proof components (Left, Right, VLeft, VRight) may contain infinity
// points in edge cases (zero witnesses, identity generators), which are valid.
func TestValidateRangeProof(t *testing.T) {
	curves := []mathlib.CurveID{
		mathlib.BLS12_381_BBS,
		mathlib.BN254,
	}

	for _, curveID := range curves {
		t.Run(fmt.Sprintf("%d", curveID), func(t *testing.T) {
			curve := mathlib.Curves[curveID]

			proof := &CspRangeProof{
				pComm: curve.GenG1,
				pokV: pokCommitment{
					A: curve.GenG1,
					Z: []*mathlib.Zr{curve.NewZrFromInt(1), curve.NewZrFromInt(2)},
				},
				u:     curve.NewZrFromInt(1),
				sComm: curve.GenG1,
				sEval: curve.NewZrFromInt(1),
				cspProof: CSPProof{
					Curve:  curve,
					Left:   []*mathlib.G1{curve.GenG1},
					Right:  []*mathlib.G1{curve.GenG1},
					VLeft:  []*mathlib.Zr{curve.NewZrFromInt(1)},
					VRight: []*mathlib.Zr{curve.NewZrFromInt(1)},
				},
			}

			t.Run("ValidProof", func(t *testing.T) {
				// Verify that a properly constructed range proof passes all validation checks
				err := validateRangeProof(proof)
				require.NoError(t, err)
			})

			t.Run("NilProof", func(t *testing.T) {
				// Verify that nil proof is rejected to prevent nil pointer dereferences
				err := validateRangeProof(nil)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "proof cannot be nil")
			})

			t.Run("NilPComm", func(t *testing.T) {
				// Verify that nil polynomial commitment is rejected (critical proof component)
				proofCopy := *proof
				proofCopy.pComm = nil
				err := validateRangeProof(&proofCopy)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "pComm cannot be nil")
			})
		})
	}
}
