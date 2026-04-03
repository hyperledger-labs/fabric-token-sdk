/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import "errors"

// Validation error types for CSP operations.
// These typed errors allow callers to distinguish between different validation
// failures using errors.Is() for better error handling and testing.
var (
	// ErrNilCurve indicates that a required curve parameter is nil.
	ErrNilCurve = errors.New("curve cannot be nil")

	// ErrInvalidCurve indicates that the curve is invalid or unsupported.
	ErrInvalidCurve = errors.New("invalid curve")

	// ErrNilCommitment indicates that a required commitment is nil.
	ErrNilCommitment = errors.New("commitment cannot be nil")

	// ErrNilValue indicates that a required value is nil.
	ErrNilValue = errors.New("value cannot be nil")

	// ErrNilRandomness indicates that required randomness is nil.
	ErrNilRandomness = errors.New("randomness cannot be nil")

	// ErrNilProof indicates that a proof parameter is nil.
	ErrNilProof = errors.New("proof cannot be nil")

	// ErrInvalidLength indicates that a slice has an incorrect length.
	ErrInvalidLength = errors.New("invalid length")

	// ErrNilElement indicates that a slice contains a nil element.
	ErrNilElement = errors.New("element cannot be nil")

	// ErrInvalidBitCount indicates that the number of bits is invalid.
	ErrInvalidBitCount = errors.New("invalid number of bits")

	// ErrWrongCurveID indicates that an element belongs to a different curve.
	ErrWrongCurveID = errors.New("wrong curve ID")
)
