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

// TestLagrangeCacheCorrectness verifies that cached denominator inverses produce
// the same Lagrange multipliers as the uncached path.
func TestLagrangeCacheCorrectness(t *testing.T) {
	curves := []math.CurveID{math.BN254, math.BLS12_381_BBS_GURVY}
	ns := []uint64{4, 8, 16}

	for _, curveID := range curves {
		curve := math.Curves[curveID]
		rand, err := curve.Rand()
		require.NoError(t, err)

		for _, n := range ns {
			c := curve.NewRandomZr(rand)

			// Cached path (via nativeLagrangeMultipliers which uses the cache).
			cached, ok, err := nativeLagrangeMultipliers(n, c, curve)
			require.NoError(t, err)
			require.True(t, ok)

			// Fallback path (big.Int based, no cache).
			fallback, err := getLagrangeMultipliers(n, c, curve)
			require.NoError(t, err)

			require.Len(t, cached, len(fallback))
			for i := range cached {
				assert.True(t, cached[i].Equals(fallback[i]),
					"curve=%d n=%d i=%d: cached and fallback results differ", curveID, n, i)
			}
		}
	}
}

// TestLagrangeCacheCorrectnessPartial verifies cached partial Lagrange multipliers.
func TestLagrangeCacheCorrectnessPartial(t *testing.T) {
	curves := []math.CurveID{math.BN254, math.BLS12_381_BBS_GURVY}
	ns := []uint64{4, 8, 16}

	for _, curveID := range curves {
		curve := math.Curves[curveID]
		rand, err := curve.Rand()
		require.NoError(t, err)

		for _, n := range ns {
			c := curve.NewRandomZr(rand)

			cached, ok, err := nativeLagrangeMultipliersPartial(n, c, curve)
			require.NoError(t, err)
			require.True(t, ok)

			fallback, err := getLagrangeMultipliersPartial(n, c, curve)
			require.NoError(t, err)

			require.Len(t, cached, len(fallback))
			for i := range cached {
				assert.True(t, cached[i].Equals(fallback[i]),
					"curve=%d n=%d i=%d: cached and fallback partial results differ", curveID, n, i)
			}
		}
	}
}

// TestLagrangeCacheIdempotent verifies that repeated cache lookups return
// identical results (cache hit path is correct).
func TestLagrangeCacheIdempotent(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	n := uint64(8)
	c := curve.NewRandomZr(rand)

	r1, ok1, err := nativeLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok1)

	r2, ok2, err := nativeLagrangeMultipliers(n, c, curve)
	require.NoError(t, err)
	require.True(t, ok2)

	require.Len(t, r1, len(r2))
	for i := range r1 {
		assert.True(t, r1[i].Equals(r2[i]), "repeated calls should return identical results at index %d", i)
	}
}

// BenchmarkLagrangeMultipliersCached benchmarks the cached path (warm cache).
func BenchmarkLagrangeMultipliersCached(b *testing.B) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(b, err)

	n := uint64(32)
	c := curve.NewRandomZr(rand)

	// Warm the cache.
	_, _, _ = nativeLagrangeMultipliers(n, c, curve)

	b.ResetTimer()
	for range b.N {
		c = curve.NewRandomZr(rand)
		_, _, _ = nativeLagrangeMultipliers(n, c, curve)
	}
}

// BenchmarkLagrangeMultipliersColdCache benchmarks the first call (cold cache, denom computation included).
func BenchmarkLagrangeMultipliersColdCache(b *testing.B) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(b, err)

	n := uint64(32)

	b.ResetTimer()
	for range b.N {
		// Clear cache entry to simulate cold start.
		lagrangeDenomCache.Delete(lagrangeCacheKey{curveID: math.BN254, n: n, partial: false})
		c := curve.NewRandomZr(rand)
		_, _, _ = nativeLagrangeMultipliers(n, c, curve)
	}
}
