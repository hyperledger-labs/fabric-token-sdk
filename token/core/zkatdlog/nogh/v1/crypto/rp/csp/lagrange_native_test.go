/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"math/big"
	"testing"

	math "github.com/IBM/mathlib"
	bls12381fr "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	bn254fr "github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNativeLagrangeMultipliersBLS12381 verifies native implementation for BLS12-381.
func TestNativeLagrangeMultipliersBLS12381(t *testing.T) {
	curve := math.Curves[math.BLS12_381_BBS_GURVY]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(4)
	c := curve.NewRandomZr(rand)

	result, ok, err := nativeLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok, "BLS12-381 should use native implementation")
	require.Len(t, result, int(n)+1)

	// Verify result matches fallback implementation
	fallback, err := getLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)
	require.Len(t, fallback, len(result))

	for i := range result {
		assert.True(t, result[i].Equals(fallback[i]),
			"native and fallback implementations should match at index %d", i)
	}
}

// TestNativeLagrangeMultipliersBN254 verifies native implementation for BN254.
func TestNativeLagrangeMultipliersBN254(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(4)
	c := curve.NewRandomZr(rand)

	result, ok, err := nativeLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok, "BN254 should use native implementation")
	require.Len(t, result, int(n)+1)

	// Verify result matches fallback implementation
	fallback, err := getLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)

	for i := range result {
		assert.True(t, result[i].Equals(fallback[i]),
			"native and fallback implementations should match at index %d", i)
	}
}

// TestNativeLagrangeMultipliersPartialBLS12381 verifies partial native implementation.
func TestNativeLagrangeMultipliersPartialBLS12381(t *testing.T) {
	curve := math.Curves[math.BLS12_381_BBS_GURVY]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(4)
	c := curve.NewRandomZr(rand)

	result, ok, err := nativeLagrangeMultipliersPartial(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok, "BLS12-381 should use native implementation")
	require.Len(t, result, int(n)+1)

	// Verify result matches fallback
	fallback, err := getLagrangeMultipliersPartial(n, c, curve)
	require.NoError(t, err)

	for i := range result {
		assert.True(t, result[i].Equals(fallback[i]),
			"native and fallback implementations should match at index %d", i)
	}
}

// TestNativeLagrangeMultipliersPartialBN254 verifies partial native implementation for BN254.
func TestNativeLagrangeMultipliersPartialBN254(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(4)
	c := curve.NewRandomZr(rand)

	result, ok, err := nativeLagrangeMultipliersPartial(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok, "BN254 should use native implementation")
	require.Len(t, result, int(n)+1)

	// Verify result matches fallback
	fallback, err := getLagrangeMultipliersPartial(n, c, curve)
	require.NoError(t, err)

	for i := range result {
		assert.True(t, result[i].Equals(fallback[i]),
			"native and fallback implementations should match at index %d", i)
	}
}

// TestNativeInterpolateBLS12381 verifies native interpolation for BLS12-381.
func TestNativeInterpolateBLS12381(t *testing.T) {
	curve := math.Curves[math.BLS12_381_BBS_GURVY]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(4)
	vals := make([]*math.Zr, n+1)
	for i := range vals {
		vals[i] = curve.NewRandomZr(rand)
	}

	result, ok, err := nativeInterpolate(n, vals, curve)
	require.NoError(t, err)
	require.True(t, ok, "BLS12-381 should use native implementation")
	require.Len(t, result, 2*int(n)+1)

	// Verify first n+1 values are unchanged
	for i := uint64(0); i <= n; i++ {
		assert.True(t, result[i].Equals(vals[i]),
			"first n+1 values should be unchanged at index %d", i)
	}

	// Verify result matches fallback
	fallback, err := interpolate(n, vals, curve)
	require.NoError(t, err)

	for i := range result {
		assert.True(t, result[i].Equals(fallback[i]),
			"native and fallback implementations should match at index %d", i)
	}
}

// TestNativeInterpolateBN254 verifies native interpolation for BN254.
func TestNativeInterpolateBN254(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(4)
	vals := make([]*math.Zr, n+1)
	for i := range vals {
		vals[i] = curve.NewRandomZr(rand)
	}

	result, ok, err := nativeInterpolate(n, vals, curve)
	require.NoError(t, err)
	require.True(t, ok, "BN254 should use native implementation")
	require.Len(t, result, 2*int(n)+1)

	// Verify first n+1 values are unchanged
	for i := uint64(0); i <= n; i++ {
		assert.True(t, result[i].Equals(vals[i]),
			"first n+1 values should be unchanged at index %d", i)
	}

	// Verify result matches fallback
	fallback, err := interpolate(n, vals, curve)
	require.NoError(t, err)

	for i := range result {
		assert.True(t, result[i].Equals(fallback[i]),
			"native and fallback implementations should match at index %d", i)
	}
}

// TestNativeFromZrToZrRoundTrip verifies conversion round-trip for BLS12-381.
func TestNativeFromZrToZrRoundTripBLS12381(t *testing.T) {
	curve := math.Curves[math.BLS12_381_BBS_GURVY]
	rand, err := curve.Rand()
	require.NoError(t, err)

	original := curve.NewRandomZr(rand)

	// Convert to native and back
	switch curve.GroupOrder.CurveID() {
	case math.BLS12_381, math.BLS12_381_GURVY, math.BLS12_381_BBS, math.BLS12_381_BBS_GURVY:
		native := nativeFromZr[bls12381fr.Element, *bls12381fr.Element](original)
		recovered := nativeToZr[bls12381fr.Element, *bls12381fr.Element](native, curve)
		assert.True(t, original.Equals(recovered), "round-trip conversion should preserve value")
	}
}

// TestNativeFromZrToZrRoundTripBN254 verifies conversion round-trip for BN254.
func TestNativeFromZrToZrRoundTripBN254(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	original := curve.NewRandomZr(rand)

	// Convert to native and back
	switch curve.GroupOrder.CurveID() {
	case math.BN254:
		native := nativeFromZr[bn254fr.Element, *bn254fr.Element](original)
		recovered := nativeToZr[bn254fr.Element, *bn254fr.Element](native, curve)
		assert.True(t, original.Equals(recovered), "round-trip conversion should preserve value")
	}
}

// TestNativeBatchInverseBLS12381 verifies batch inversion for BLS12-381.
func TestNativeBatchInverseBLS12381(t *testing.T) {
	curve := math.Curves[math.BLS12_381_BBS_GURVY]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := 10
	elems := make([]*bls12381fr.Element, n)
	for i := range elems {
		zr := curve.NewRandomZr(rand)
		elems[i] = nativeFromZr[bls12381fr.Element, *bls12381fr.Element](zr)
	}

	invs := nativeBatchInverse[bls12381fr.Element, *bls12381fr.Element](elems)
	require.Len(t, invs, n)

	// Verify each inverse
	for i := range n {
		var prod bls12381fr.Element
		prod.Mul(elems[i], invs[i])
		
		var one bls12381fr.Element
		one.SetInt64(1)
		
		assert.True(t, prod.Equal(&one), "element * inverse should equal 1 at index %d", i)
	}
}

// TestNativeBatchInverseBN254 verifies batch inversion for BN254.
func TestNativeBatchInverseBN254(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := 10
	elems := make([]*bn254fr.Element, n)
	for i := range elems {
		zr := curve.NewRandomZr(rand)
		elems[i] = nativeFromZr[bn254fr.Element, *bn254fr.Element](zr)
	}

	invs := nativeBatchInverse[bn254fr.Element, *bn254fr.Element](elems)
	require.Len(t, invs, n)

	// Verify each inverse
	for i := range n {
		var prod bn254fr.Element
		prod.Mul(elems[i], invs[i])
		
		var one bn254fr.Element
		one.SetInt64(1)
		
		assert.True(t, prod.Equal(&one), "element * inverse should equal 1 at index %d", i)
	}
}

// TestNativeBatchInverseEmpty verifies batch inversion with empty input.
func TestNativeBatchInverseEmpty(t *testing.T) {
	var elems []*bls12381fr.Element
	invs := nativeBatchInverse[bls12381fr.Element, *bls12381fr.Element](elems)
	assert.Nil(t, invs, "batch inverse of empty slice should return nil")
}

// TestNativeBatchInverseSingle verifies batch inversion with single element.
func TestNativeBatchInverseSingle(t *testing.T) {
	curve := math.Curves[math.BLS12_381_BBS_GURVY]
	rand, err := curve.Rand()
	require.NoError(t, err)

	zr := curve.NewRandomZr(rand)
	elem := nativeFromZr[bls12381fr.Element, *bls12381fr.Element](zr)
	elems := []*bls12381fr.Element{elem}

	invs := nativeBatchInverse[bls12381fr.Element, *bls12381fr.Element](elems)
	require.Len(t, invs, 1)

	var prod bls12381fr.Element
	prod.Mul(elem, invs[0])
	
	var one bls12381fr.Element
	one.SetInt64(1)
	
	assert.True(t, prod.Equal(&one), "element * inverse should equal 1")
}

// TestNativeConversionZeroValue verifies conversion of zero value.
func TestNativeConversionZeroValue(t *testing.T) {
	curve := math.Curves[math.BN254]
	zero := curve.NewZrFromInt(0)

	native := nativeFromZr[bn254fr.Element, *bn254fr.Element](zero)
	recovered := nativeToZr[bn254fr.Element, *bn254fr.Element](native, curve)

	assert.True(t, zero.Equals(recovered), "zero value should round-trip correctly")
}

// TestNativeConversionOneValue verifies conversion of one value.
func TestNativeConversionOneValue(t *testing.T) {
	curve := math.Curves[math.BN254]
	one := curve.NewZrFromInt(1)

	native := nativeFromZr[bn254fr.Element, *bn254fr.Element](one)
	recovered := nativeToZr[bn254fr.Element, *bn254fr.Element](native, curve)

	assert.True(t, one.Equals(recovered), "one value should round-trip correctly")
}

// TestNativeConversionLargeValue verifies conversion of large field element.
func TestNativeConversionLargeValue(t *testing.T) {
	curve := math.Curves[math.BN254]
	
	// Create a large value close to field order
	orderBytes := curve.GroupOrder.Bytes()
	largeInt := new(big.Int).SetBytes(orderBytes)
	largeInt.Sub(largeInt, big.NewInt(1))
	large := curve.NewZrFromBytes(largeInt.Bytes())

	native := nativeFromZr[bn254fr.Element, *bn254fr.Element](large)
	recovered := nativeToZr[bn254fr.Element, *bn254fr.Element](native, curve)

	assert.True(t, large.Equals(recovered), "large value should round-trip correctly")
}

// TestNativeLagrangeMultipliersConsistency verifies consistency across multiple calls.
func TestNativeLagrangeMultipliersConsistency(t *testing.T) {
	curve := math.Curves[math.BLS12_381_BBS_GURVY]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(4)
	c := curve.NewRandomZr(rand)

	// Call multiple times
	result1, ok1, err := nativeLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok1)

	result2, ok2, err := nativeLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok2)

	// Results should be identical
	require.Len(t, result1, len(result2))
	for i := range result1 {
		assert.True(t, result1[i].Equals(result2[i]),
			"multiple calls should produce identical results at index %d", i)
	}
}

// TestNativeInterpolateConsistency verifies interpolation consistency.
func TestNativeInterpolateConsistency(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(3)
	vals := make([]*math.Zr, n+1)
	for i := range vals {
		vals[i] = curve.NewRandomZr(rand)
	}

	// Call multiple times
	result1, ok1, err := nativeInterpolate(n, vals, curve)
	require.NoError(t, err)
	require.True(t, ok1)

	result2, ok2, err := nativeInterpolate(n, vals, curve)
	require.NoError(t, err)
	require.True(t, ok2)

	// Results should be identical
	require.Len(t, result1, len(result2))
	for i := range result1 {
		assert.True(t, result1[i].Equals(result2[i]),
			"multiple calls should produce identical results at index %d", i)
	}
}
