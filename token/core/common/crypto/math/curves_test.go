/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package math

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/IBM/mathlib/driver/gurvy/bls12381"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurveWithFastRNG(t *testing.T) {
	c := NewCurveWithFastRNG(bls12381.NewBBSCurve())
	assert.NotNil(t, c)
	r, err := c.Rand()
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestCurveIDToString(t *testing.T) {
	tests := []struct {
		id       math.CurveID
		expected string
	}{
		{math.FP256BN_AMCL, "FP256BN_AMCL"},
		{math.BN254, "BN254"},
		{math.FP256BN_AMCL_MIRACL, "FP256BN_AMCL_MIRACL"},
		{math.BLS12_381, "BLS12_381"},
		{math.BLS12_377_GURVY, "BLS12_377_GURVY"},
		{math.BLS12_381_GURVY, "BLS12_381_GURVY"},
		{math.BLS12_381_BBS, "BLS12_381_BBS"},
		{math.BLS12_381_BBS_GURVY, "BLS12_381_BBS_GURVY"},
		{BLS12_381_BBS_GURVY_FAST_RNG, "BLS12_381_BBS_GURVY_FAST_RNG"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, CurveIDToString(tt.id))
	}

	assert.Panics(t, func() {
		CurveIDToString(math.CurveID(9999))
	})
}

func TestStringToCurveID(t *testing.T) {
	tests := []struct {
		s        string
		expected math.CurveID
	}{
		{"FP256BN_AMCL", math.FP256BN_AMCL},
		{"BN254", math.BN254},
		{"FP256BN_AMCL_MIRACL", math.FP256BN_AMCL_MIRACL},
		{"BLS12_381_BBS", math.BLS12_381_BBS},
		{"BLS12_381_BBS_GURVY", math.BLS12_381_BBS_GURVY},
		{"BLS12_381_BBS_GURVY_FAST_RNG", BLS12_381_BBS_GURVY_FAST_RNG},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, StringToCurveID(tt.s))
	}

	assert.Panics(t, func() {
		StringToCurveID("unknown")
	})
}
