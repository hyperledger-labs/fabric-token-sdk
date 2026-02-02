/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package math

import (
	"testing"

	mathlib "github.com/IBM/mathlib"
	"github.com/stretchr/testify/require"
)

func TestZeroOneTwoAndCached(t *testing.T) {
	c := mathlib.Curves[mathlib.BN254]
	require.NotNil(t, c)

	// basic values
	zrZero := Zero(c)
	zrOne := One(c)
	zrTwo := Two(c)

	zrEquals(t, zrZero, c.NewZrFromUint64(0))
	zrEquals(t, zrOne, c.NewZrFromUint64(1))
	zrEquals(t, zrTwo, c.NewZrFromUint64(2))

	// cache hit for a small index
	vSnapshot, pSnapshot, sSnapshot := snapshotCaches()
	defer restoreCaches(vSnapshot, pSnapshot, sSnapshot)

	// ensure cache contains 3
	valMap := valueCache[c.ID()]
	require.NotNil(t, valMap)
	v, ok := valMap[3]
	require.True(t, ok)
	rz := NewCachedZrFromInt(c, 3)
	zrEquals(t, rz, v)
}

func TestPowerOfTwoAndSum(t *testing.T) {
	c := mathlib.Curves[mathlib.BN254]
	require.NotNil(t, c)

	vSnapshot, pSnapshot, sSnapshot := snapshotCaches()
	defer restoreCaches(vSnapshot, pSnapshot, sSnapshot)

	// test cached powers
	pc := powerCache[c.ID()]
	require.NotNil(t, pc)
	p0, ok := pc[0]
	require.True(t, ok)
	zrEquals(t, p0, PowerOfTwo(c, 0))

	// cached sum
	sc := sumOfPowerCache[c.ID()]
	require.NotNil(t, sc)
	// sum for n=1 should equal 2^0
	s1, ok := sc[1]
	require.True(t, ok)
	zrEquals(t, s1, SumOfPowersOfTwo(c, 1))

	// compute a larger power not in cache (use an index beyond NumBits)
	bigIndex := uint64(NumBits + 5)
	computed := c.NewZrFromUint64(2).PowMod(c.NewZrFromUint64(bigIndex))
	zrEquals(t, computed, PowerOfTwo(c, bigIndex))

	// compute sum iteratively
	n := uint64(10)
	iter := c.NewZrFromUint64(0)
	for i := uint64(0); i < n; i++ {
		p := c.NewZrFromUint64(2).PowMod(c.NewZrFromUint64(i))
		iter = c.ModAdd(iter, p, c.GroupOrder)
	}
	// compare with SumOfPowersOfTwo if present in cache; if not, ensure SumOfPowersOfTwo panics
	if _, ok := sumOfPowerCache[c.ID()][n]; ok {
		zrEquals(t, iter, SumOfPowersOfTwo(c, n))
	} else {
		require.Panics(t, func() { SumOfPowersOfTwo(c, n) })
	}
}

// helper: compare Zr values using Equals
func zrEquals(t *testing.T, a, b *mathlib.Zr) {
	t.Helper()
	require.NotNil(t, a)
	require.NotNil(t, b)
	// mathlib.Zr provides an Equals method
	require.True(t, a.Equals(b))
}

// snapshotCaches creates shallow copies of the package caches so tests can
// modify the global maps and restore them afterwards.
func snapshotCaches() (map[mathlib.CurveID]map[uint64]*mathlib.Zr, map[mathlib.CurveID]map[uint64]*mathlib.Zr, map[mathlib.CurveID]map[uint64]*mathlib.Zr) {
	v := make(map[mathlib.CurveID]map[uint64]*mathlib.Zr)
	p := make(map[mathlib.CurveID]map[uint64]*mathlib.Zr)
	s := make(map[mathlib.CurveID]map[uint64]*mathlib.Zr)
	for k, vv := range valueCache {
		m := make(map[uint64]*mathlib.Zr, len(vv))
		for kk, v := range vv {
			m[kk] = v
		}
		v[k] = m
	}
	for k, vv := range powerCache {
		m := make(map[uint64]*mathlib.Zr, len(vv))
		for kk, v := range vv {
			m[kk] = v
		}
		p[k] = m
	}
	for k, vv := range sumOfPowerCache {
		m := make(map[uint64]*mathlib.Zr, len(vv))
		for kk, v := range vv {
			m[kk] = v
		}
		s[k] = m
	}
	return v, p, s
}

func restoreCaches(v, p, s map[mathlib.CurveID]map[uint64]*mathlib.Zr) {
	valueCache = v
	powerCache = p
	sumOfPowerCache = s
}
