/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"errors"
	"fmt"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
)

// validateCurve checks if the curve is non-nil and supported.
func validateCurve(curve *mathlib.Curve) error {
	if curve == nil {
		return ErrNilCurve
	}
	if curve.GroupOrder == nil {
		return fmt.Errorf("%w: group order cannot be nil", ErrInvalidCurve)
	}
	if curve.GenG1 == nil {
		return fmt.Errorf("%w: generator G1 cannot be nil", ErrInvalidCurve)
	}

	return nil
}

// validateG1Slice checks if a slice of G1 elements is valid.
// Note: This validation allows infinity points (identity elements) which can legitimately
// appear in generators or proof elements for edge cases.
func validateG1Slice(name string, elements []*mathlib.G1, curve *mathlib.Curve, expectedLen int) error {
	if elements == nil {
		return fmt.Errorf("%s cannot be nil", name)
	}
	if expectedLen > 0 && len(elements) != expectedLen {
		return fmt.Errorf("%w: %s expected %d, got %d", ErrInvalidLength, name, expectedLen, len(elements))
	}
	// Check each element for nil and curve ID match, but allow infinity
	for i, elem := range elements {
		if elem == nil {
			return fmt.Errorf("%w: %s[%d]", ErrNilElement, name, i)
		}
		if elem.CurveID() != curve.ID() {
			return fmt.Errorf("%w: %s[%d]", ErrWrongCurveID, name, i)
		}
	}

	return nil
}

// validateZrSlice checks if a slice of Zr elements is valid using the math package utilities.
func validateZrSlice(name string, elements []*mathlib.Zr, curve *mathlib.Curve, expectedLen int) error {
	if elements == nil {
		return fmt.Errorf("%s cannot be nil", name)
	}
	if expectedLen > 0 && len(elements) != expectedLen {
		return fmt.Errorf("%w: %s expected %d, got %d", ErrInvalidLength, name, expectedLen, len(elements))
	}
	if err := math.CheckZrElements(elements, curve.ID(), uint64(len(elements))); err != nil {
		return fmt.Errorf("%s validation failed: %w", name, err)
	}

	return nil
}

// validateCSPProverInputs validates all inputs for CSP prover.
func validateCSPProverInputs(curve *mathlib.Curve, p *cspProver) error {
	if err := validateCurve(curve); err != nil {
		return fmt.Errorf("invalid curve: %w", err)
	}
	if p.Commitment == nil {
		return ErrNilCommitment
	}
	if p.Value == nil {
		return ErrNilValue
	}

	expected := 1 << p.NumberOfRounds
	if err := validateG1Slice("generators", p.Generators, curve, expected); err != nil {
		return err
	}
	if err := validateZrSlice("linear form", p.LinearForm, curve, expected); err != nil {
		return err
	}
	if err := validateZrSlice("witness", p.witness, curve, expected); err != nil {
		return err
	}

	return nil
}

// validateCSPVerifierInputs validates all inputs for CSP verifier.
func validateCSPVerifierInputs(curve *mathlib.Curve, v *cspVerifier) error {
	if err := validateCurve(curve); err != nil {
		return fmt.Errorf("invalid curve: %w", err)
	}
	if v.Commitment == nil {
		return ErrNilCommitment
	}
	if v.Value == nil {
		return ErrNilValue
	}

	expected := 1 << v.NumberOfRounds
	if err := validateG1Slice("generators", v.Generators, curve, expected); err != nil {
		return err
	}
	if err := validateZrSlice("linear form", v.LinearForm, curve, expected); err != nil {
		return err
	}

	return nil
}

// validateCSPProof validates the structure of a CSP proof.
// Note: Proof elements (Left, Right) are NOT checked for infinity because
// infinity points can legitimately appear in proofs for edge cases like zero witnesses.
func validateCSPProof(curve *mathlib.Curve, proof *CSPProof, expectedRounds uint64) error {
	if proof == nil {
		return ErrNilProof
	}
	if err := validateCurve(curve); err != nil {
		return fmt.Errorf("invalid proof curve: %w", err)
	}
	// Validate proof arrays without strict infinity checks (proof elements can be infinity)
	if proof.Left == nil {
		return errors.New("proof.Left cannot be nil")
	}
	// Validate proof arrays length - use uint64 comparison to avoid conversion
	if uint64(len(proof.Left)) != expectedRounds {
		return fmt.Errorf("%w: proof.Left expected %d, got %d", ErrInvalidLength, expectedRounds, len(proof.Left))
	}
	if proof.Right == nil {
		return errors.New("proof.Right cannot be nil")
	}
	if uint64(len(proof.Right)) != expectedRounds {
		return fmt.Errorf("%w: proof.Right expected %d, got %d", ErrInvalidLength, expectedRounds, len(proof.Right))
	}
	for i, elem := range proof.Left {
		if elem == nil {
			return fmt.Errorf("%w: proof.Left[%d]", ErrNilElement, i)
		}
		if elem.CurveID() != curve.ID() {
			return fmt.Errorf("%w: proof.Left[%d]", ErrWrongCurveID, i)
		}
	}
	for i, elem := range proof.Right {
		if elem == nil {
			return fmt.Errorf("%w: proof.Right[%d]", ErrNilElement, i)
		}
		if elem.CurveID() != curve.ID() {
			return fmt.Errorf("%w: proof.Right[%d]", ErrWrongCurveID, i)
		}
	}
	// For Zr slices, validate length using uint64 comparison
	if uint64(len(proof.VLeft)) != expectedRounds {
		return fmt.Errorf("%w: proof.VLeft expected %d, got %d", ErrInvalidLength, expectedRounds, len(proof.VLeft))
	}
	if uint64(len(proof.VRight)) != expectedRounds {
		return fmt.Errorf("%w: proof.VRight expected %d, got %d", ErrInvalidLength, expectedRounds, len(proof.VRight))
	}
	if err := math.CheckZrElements(proof.VLeft, curve.ID(), expectedRounds); err != nil {
		return fmt.Errorf("proof.VLeft validation failed: %w", err)
	}
	if err := math.CheckZrElements(proof.VRight, curve.ID(), expectedRounds); err != nil {
		return fmt.Errorf("proof.VRight validation failed: %w", err)
	}

	return nil
}

// validateRangeProverInputs validates all inputs for range proof prover.
func validateRangeProverInputs(curve *mathlib.Curve, p *cspRangeProver) error {
	if err := validateCurve(curve); err != nil {
		return fmt.Errorf("invalid curve: %w", err)
	}
	if p.VCommitment == nil {
		return ErrNilCommitment
	}
	if p.v == nil {
		return ErrNilValue
	}
	if p.r == nil {
		return ErrNilRandomness
	}
	if p.NumberOfBits == 0 {
		return fmt.Errorf("%w: must be greater than 0", ErrInvalidBitCount)
	}
	if p.NumberOfBits > 64 {
		return fmt.Errorf("%w: cannot exceed 64", ErrInvalidBitCount)
	}

	if err := validateG1Slice("VGenerators", p.VGenerators, curve, 2); err != nil {
		return err
	}
	if err := validateG1Slice("AGenerators", p.AGenerators, curve, int(p.NumberOfBits+1)); err != nil {
		return err
	}
	if err := validateG1Slice("BGenerators", p.BGenerators, curve, int(p.NumberOfBits+1)); err != nil {
		return err
	}

	return nil
}

// validateRangeVerifierInputs validates all inputs for range proof verifier.
func validateRangeVerifierInputs(curve *mathlib.Curve, v *cspRangeVerifier) error {
	if err := validateCurve(curve); err != nil {
		return fmt.Errorf("invalid curve: %w", err)
	}
	if v.VCommitment == nil {
		return ErrNilCommitment
	}
	if v.NumberOfBits == 0 {
		return fmt.Errorf("%w: must be greater than 0", ErrInvalidBitCount)
	}
	if v.NumberOfBits > 64 {
		return fmt.Errorf("%w: cannot exceed 64", ErrInvalidBitCount)
	}

	if err := validateG1Slice("VGenerators", v.VGenerators, curve, 2); err != nil {
		return err
	}
	if err := validateG1Slice("AGenerators", v.AGenerators, curve, int(v.NumberOfBits+1)); err != nil {
		return err
	}
	if err := validateG1Slice("BGenerators", v.BGenerators, curve, int(v.NumberOfBits+1)); err != nil {
		return err
	}

	return nil
}

// validateRangeProof validates the structure of a range proof.
func validateRangeProof(curve *mathlib.Curve, proof *CspRangeProof) error {
	if proof == nil {
		return ErrNilProof
	}
	if err := validateCurve(curve); err != nil {
		return fmt.Errorf("invalid curve: %w", err)
	}
	if proof.pComm == nil {
		return errors.New("proof.pComm cannot be nil")
	}
	if proof.pComm.CurveID() != curve.ID() {
		return fmt.Errorf("%w: proof.pComm", ErrWrongCurveID)
	}
	if proof.pokV.A == nil {
		return errors.New("proof.pokV.A cannot be nil")
	}
	if proof.pokV.A.CurveID() != curve.ID() {
		return fmt.Errorf("%w: proof.pokV.A", ErrWrongCurveID)
	}
	if err := validateZrSlice("proof.pokV.Z", proof.pokV.Z, curve, 2); err != nil {
		return err
	}
	if proof.u == nil {
		return errors.New("proof.u cannot be nil")
	}
	if proof.sComm == nil {
		return errors.New("proof.sComm cannot be nil")
	}
	if proof.sComm.CurveID() != curve.ID() {
		return fmt.Errorf("%w: proof.sComm", ErrWrongCurveID)
	}
	if proof.sEval == nil {
		return errors.New("proof.sEval cannot be nil")
	}

	return validateCSPProof(curve, &proof.cspProof, uint64(len(proof.cspProof.Left)))
}
