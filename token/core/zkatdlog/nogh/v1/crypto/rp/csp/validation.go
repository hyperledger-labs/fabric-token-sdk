/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
)

// validateCurve checks if the curve is non-nil and supported.
func validateCurve(curve *mathlib.Curve) error {
	if curve == nil {
		return ErrNilCurve
	}
	if curve.GroupOrder == nil {
		return errors.Wrapf(ErrInvalidCurve, "group order cannot be nil")
	}
	if curve.GenG1 == nil {
		return errors.Wrapf(ErrInvalidCurve, "generator G1 cannot be nil")
	}

	return nil
}

// validateG1Slice checks if a slice of G1 elements is valid.
// Note: This validation allows infinity points (identity elements) which can legitimately
// appear in generators or proof elements for edge cases.
func validateG1Slice(name string, elements []*mathlib.G1, curve *mathlib.Curve, expectedLen int) error {
	if elements == nil {
		return errors.Errorf("%s cannot be nil", name)
	}
	if expectedLen > 0 && len(elements) != expectedLen {
		return errors.Wrapf(ErrInvalidLength, "%s expected %d, got %d", name, expectedLen, len(elements))
	}
	// Check each element for nil and curve ID match, but allow infinity
	for i, elem := range elements {
		if elem == nil {
			return errors.Wrapf(ErrNilElement, "%s[%d]", name, i)
		}
		if elem.CurveID() != curve.ID() {
			return errors.Wrapf(ErrWrongCurveID, "%s[%d]", name, i)
		}
	}

	return nil
}

// validateZrSlice checks if a slice of Zr elements is valid using the math package utilities.
func validateZrSlice(name string, elements []*mathlib.Zr, curve *mathlib.Curve, expectedLen int) error {
	if elements == nil {
		return errors.Errorf("%s cannot be nil", name)
	}
	if expectedLen > 0 && len(elements) != expectedLen {
		return errors.Wrapf(ErrInvalidLength, "%s expected %d, got %d", name, expectedLen, len(elements))
	}
	if err := math.CheckZrElements(elements, curve.ID(), uint64(len(elements))); err != nil {
		return errors.Wrapf(err, "%s validation failed", name)
	}

	return nil
}

// validateCSPProverInputs validates all inputs for CSP prover.
func validateCSPProverInputs(curve *mathlib.Curve, p *cspProver) error {
	if err := validateCurve(curve); err != nil {
		return errors.Wrapf(err, "invalid curve")
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
		return errors.Wrapf(err, "invalid curve")
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
		return errors.Wrapf(err, "invalid proof curve")
	}
	// Validate proof arrays without strict infinity checks (proof elements can be infinity)
	if proof.Left == nil {
		return errors.New("proof.Left cannot be nil")
	}
	// Validate proof arrays length - use uint64 comparison to avoid conversion
	if uint64(len(proof.Left)) != expectedRounds {
		return errors.Wrapf(ErrInvalidLength, "proof.Left expected %d, got %d", expectedRounds, len(proof.Left))
	}
	if proof.Right == nil {
		return errors.New("proof.Right cannot be nil")
	}
	if uint64(len(proof.Right)) != expectedRounds {
		return errors.Wrapf(ErrInvalidLength, "proof.Right expected %d, got %d", expectedRounds, len(proof.Right))
	}
	for i, elem := range proof.Left {
		if elem == nil {
			return errors.Wrapf(ErrNilElement, "proof.Left[%d]", i)
		}
		if elem.CurveID() != curve.ID() {
			return errors.Wrapf(ErrWrongCurveID, "proof.Left[%d]", i)
		}
	}
	for i, elem := range proof.Right {
		if elem == nil {
			return errors.Wrapf(ErrNilElement, "proof.Right[%d]", i)
		}
		if elem.CurveID() != curve.ID() {
			return errors.Wrapf(ErrWrongCurveID, "proof.Right[%d]", i)
		}
	}
	// For Zr slices, validate length using uint64 comparison
	if uint64(len(proof.VLeft)) != expectedRounds {
		return errors.Wrapf(ErrInvalidLength, "proof.VLeft expected %d, got %d", expectedRounds, len(proof.VLeft))
	}
	if uint64(len(proof.VRight)) != expectedRounds {
		return errors.Wrapf(ErrInvalidLength, "proof.VRight expected %d, got %d", expectedRounds, len(proof.VRight))
	}
	if err := math.CheckZrElements(proof.VLeft, curve.ID(), expectedRounds); err != nil {
		return errors.Wrapf(err, "proof.VLeft validation failed")
	}
	if err := math.CheckZrElements(proof.VRight, curve.ID(), expectedRounds); err != nil {
		return errors.Wrapf(err, "proof.VRight validation failed")
	}

	return nil
}

// validateRangeProverInputs validates all inputs for range proof prover.
func validateRangeProverInputs(curve *mathlib.Curve, p *cspRangeProver) error {
	if err := validateCurve(curve); err != nil {
		return errors.Wrapf(err, "invalid curve")
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
		return errors.Wrapf(ErrInvalidBitCount, "must be greater than 0")
	}
	if p.NumberOfBits > 64 {
		return errors.Wrapf(ErrInvalidBitCount, "cannot exceed 64")
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
		return errors.Wrapf(err, "invalid curve")
	}
	if v.VCommitment == nil {
		return ErrNilCommitment
	}
	if v.NumberOfBits == 0 {
		return errors.Wrapf(ErrInvalidBitCount, "must be greater than 0")
	}
	if v.NumberOfBits > 64 {
		return errors.Wrapf(ErrInvalidBitCount, "cannot exceed 64")
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
		return errors.Wrapf(err, "invalid curve")
	}
	if proof.pComm == nil {
		return errors.Wrap(ErrNilCommitment, "proof.pComm cannot be nil")
	}
	if proof.pComm.CurveID() != curve.ID() {
		return errors.Wrapf(ErrWrongCurveID, "proof.pComm")
	}
	if proof.pokV.A == nil {
		return errors.Wrap(ErrNilCommitment, "proof.pokV.A cannot be nil")
	}
	if proof.pokV.A.CurveID() != curve.ID() {
		return errors.Wrapf(ErrWrongCurveID, "proof.pokV.A")
	}
	if err := validateZrSlice("proof.pokV.Z", proof.pokV.Z, curve, 2); err != nil {
		return err
	}
	if proof.u == nil {
		return errors.Wrap(ErrNilValue, "proof.u cannot be nil")
	}
	if proof.sComm == nil {
		return errors.Wrap(ErrNilCommitment, "proof.sComm cannot be nil")
	}
	if proof.sComm.CurveID() != curve.ID() {
		return errors.Wrapf(ErrWrongCurveID, "proof.sComm")
	}
	if proof.sEval == nil {
		return errors.Wrap(ErrNilValue, "proof.sEval cannot be nil")
	}

	return validateCSPProof(curve, &proof.cspProof, uint64(len(proof.cspProof.Left)))
}
