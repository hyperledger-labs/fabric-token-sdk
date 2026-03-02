/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"math/big"

	mathlib "github.com/IBM/mathlib"
	bls12381fr "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	bn254fr "github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

// gnarkFr is the pointer constraint satisfied by gnark-crypto fr.Element types
// such as bls12-381/fr.Element and bn254/fr.Element.
//
// The convention for all methods follows gnark-crypto: the receiver is the
// output, e.g. z.Mul(x, y) sets z = x·y and returns z.
type gnarkFr[T any] interface {
	*T
	SetBigInt(*big.Int) *T
	BigInt(*big.Int) *big.Int
	Add(*T, *T) *T
	Sub(*T, *T) *T
	Mul(*T, *T) *T
	SetInt64(int64) *T
	Inverse(*T) *T
}

// nativeElem allocates and returns a new zero-valued field element of type E.
func nativeElem[T any, E gnarkFr[T]]() E { return E(new(T)) }

// nativeFromZr converts *mathlib.Zr to a native field element via big.Int.
// Only one big.Int conversion happens here.
func nativeFromZr[T any, E gnarkFr[T]](z *mathlib.Zr) E {
	e := nativeElem[T, E]()
	e.SetBigInt(new(big.Int).SetBytes(z.Bytes()))

	return e
}

// nativeToZr converts a native field element back to *mathlib.Zr via big.Int.
func nativeToZr[T any, E gnarkFr[T]](e E, curve *mathlib.Curve) *mathlib.Zr {
	var bi big.Int
	e.BigInt(&bi)

	return curve.NewZrFromBytes(bi.Bytes())
}

// nativeBatchInverse computes the modular inverse of every element in elems
// using Montgomery's batch-inversion trick: 2(n-1) multiplications + 1 inversion.
// A zero input element yields a zero output.
func nativeBatchInverse[T any, E gnarkFr[T]](elems []E) []E {
	n := len(elems)
	if n == 0 {
		return nil
	}

	// prefix[i] = elems[0] · elems[1] · … · elems[i]
	prefix := make([]T, n)
	prefix[0] = *elems[0]
	for i := 1; i < n; i++ {
		E(&prefix[i]).Mul(E(&prefix[i-1]), elems[i])
	}

	// acc = prefix[n-1]^{-1}
	var acc T
	E(&acc).Inverse(E(&prefix[n-1]))

	// Unwind: result[i] = acc · prefix[i-1], then acc ← acc · elems[i]
	result := make([]T, n)
	resultE := make([]E, n)
	for i := range result {
		resultE[i] = E(&result[i])
	}
	for i := n - 1; i > 0; i-- {
		resultE[i].Mul(E(&acc), E(&prefix[i-1]))
		E(&acc).Mul(E(&acc), elems[i])
	}
	result[0] = acc

	return resultE
}

// getLagrangeMultipliersNative is the native fr.Element implementation of
// getLagrangeMultipliers. Conversions between mathlib.Zr and fr.Element occur
// only once at the boundary (once for input c, n+1 times for the output slice),
// so the O(n²) arithmetic runs entirely in native Montgomery form.
func getLagrangeMultipliersNative[T any, E gnarkFr[T]](n uint64, c *mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, error) {
	m := int(n) + 1 // #nosec G115

	// Convert c once.
	cE := nativeFromZr[T, E](c)

	// cMinusJ[j] = c - j  for j = 0..n
	cMinusJ := make([]T, m)
	cMinusJE := make([]E, m)
	for j := range m {
		cMinusJE[j] = E(&cMinusJ[j])
		var jE T
		E(&jE).SetInt64(int64(j))
		cMinusJE[j].Sub(cE, E(&jE))
	}

	// Compute numerator and denominator for each Lagrange basis polynomial L_i(c).
	numers := make([]T, m)
	denoms := make([]T, m)
	numersE := make([]E, m)
	denomsE := make([]E, m)
	for i := range numers {
		numersE[i] = E(&numers[i])
		denomsE[i] = E(&denoms[i])
	}

	for i := range m {
		numersE[i].SetInt64(1)
		denomsE[i].SetInt64(1)
		var diff T
		diffE := E(&diff)
		for j := range m {
			if j == i {
				continue
			}
			numersE[i].Mul(numersE[i], cMinusJE[j])
			diffE.SetInt64(int64(i - j)) // handles negative i-j correctly
			denomsE[i].Mul(denomsE[i], diffE)
		}
	}

	denomInvs := nativeBatchInverse[T, E](denomsE)

	result := make([]*mathlib.Zr, m)
	for i := range m {
		var prod T
		E(&prod).Mul(numersE[i], denomInvs[i])
		result[i] = nativeToZr[T, E](E(&prod), curve)
	}

	return result, nil
}

// getLagrangeMultipliersPartialNative is the native fr.Element implementation of
// getLagrangeMultipliersPartial. Same boundary-only conversion strategy.
func getLagrangeMultipliersPartialNative[T any, E gnarkFr[T]](n uint64, c *mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, error) {
	total := 2*int(n) + 1 // #nosec G115 // all evaluation points: 0..2n

	cE := nativeFromZr[T, E](c)

	// cMinusJ[j] = c - j  for j = 0..2n
	cMinusJ := make([]T, total)
	cMinusJE := make([]E, total)
	for j := range total {
		cMinusJE[j] = E(&cMinusJ[j])
		var jE T
		E(&jE).SetInt64(int64(j))
		cMinusJE[j].Sub(cE, E(&jE))
	}

	// Relevant indices in the full point set: {0, n+1, n+2, ..., 2n}
	relevant := make([]int, int(n)+1) // #nosec G115
	relevant[0] = 0
	for k := 1; k <= int(n); k++ { // #nosec G115
		relevant[k] = int(n) + k // #nosec G115
	}

	numers := make([]T, len(relevant))
	denoms := make([]T, len(relevant))
	numersE := make([]E, len(relevant))
	denomsE := make([]E, len(relevant))
	for k := range relevant {
		numersE[k] = E(&numers[k])
		denomsE[k] = E(&denoms[k])
	}

	for k, i := range relevant {
		numersE[k].SetInt64(1)
		denomsE[k].SetInt64(1)
		var diff T
		diffE := E(&diff)
		for j := range total {
			if j == i {
				continue
			}
			numersE[k].Mul(numersE[k], cMinusJE[j])
			diffE.SetInt64(int64(i - j))
			denomsE[k].Mul(denomsE[k], diffE)
		}
	}

	denomInvs := nativeBatchInverse[T, E](denomsE)

	result := make([]*mathlib.Zr, len(relevant))
	for k := range relevant {
		var prod T
		E(&prod).Mul(numersE[k], denomInvs[k])
		result[k] = nativeToZr[T, E](E(&prod), curve)
	}

	return result, nil
}

// interpolateNative is the native fr.Element implementation of interpolate.
func interpolateNative[T any, E gnarkFr[T]](n uint64, valuesOverN []*mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, error) {
	m := int(n) + 1 // #nosec G115

	// Convert all input values to native elements once.
	vals := make([]T, m)
	valsE := make([]E, m)
	for i := range m {
		valsE[i] = E(&vals[i])
		valsE[i].SetBigInt(new(big.Int).SetBytes(valuesOverN[i].Bytes()))
	}

	// d_i = ∏_{j≠i}(i-j)  for i = 0..n  (denominator of the i-th basis polynomial)
	denoms := make([]T, m)
	denomsE := make([]E, m)
	for i := range denoms {
		denomsE[i] = E(&denoms[i])
		denomsE[i].SetInt64(1)
		var diff T
		diffE := E(&diff)
		for j := range m {
			if j == i {
				continue
			}
			diffE.SetInt64(int64(i - j))
			denomsE[i].Mul(denomsE[i], diffE)
		}
	}
	denomInvs := nativeBatchInverse[T, E](denomsE)

	// First m entries are the inputs verbatim.
	result := make([]*mathlib.Zr, 2*int(n)+1) // #nosec G115
	copy(result, valuesOverN)

	// Evaluate at each x in {n+1, ..., 2n} via Lagrange interpolation.
	for x := int(n) + 1; x <= 2*int(n); x++ { // #nosec G115
		// xMinusJ[j] = x - j, and px = ∏_j xMinusJ[j]
		xMinusJ := make([]T, m)
		xMinusJE := make([]E, m)
		var px T
		pxE := E(&px)
		pxE.SetInt64(1)
		for j := range m {
			xMinusJE[j] = E(&xMinusJ[j])
			xMinusJE[j].SetInt64(int64(x - j)) // #nosec G115
			pxE.Mul(pxE, xMinusJE[j])
		}

		xMinusJInvs := nativeBatchInverse[T, E](xMinusJE)

		var val T
		valE := E(&val)
		for i := range m {
			var li T
			liE := E(&li)
			liE.Mul(pxE, xMinusJInvs[i])
			liE.Mul(liE, denomInvs[i])
			liE.Mul(liE, valsE[i])
			valE.Add(valE, liE)
		}
		result[x] = nativeToZr[T, E](valE, curve)
	}

	return result, nil
}

// nativeLagrangeMultipliers dispatches getLagrangeMultipliers to the native
// fr.Element implementation for supported curves.
func nativeLagrangeMultipliers(n uint64, c *mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, bool, error) {
	switch curve.GroupOrder.CurveID() {
	case mathlib.BLS12_381, mathlib.BLS12_381_GURVY, mathlib.BLS12_381_BBS, mathlib.BLS12_381_BBS_GURVY:
		r, err := getLagrangeMultipliersNative[bls12381fr.Element, *bls12381fr.Element](n, c, curve)

		return r, true, err
	case mathlib.BN254:
		r, err := getLagrangeMultipliersNative[bn254fr.Element, *bn254fr.Element](n, c, curve)

		return r, true, err
	}

	return nil, false, nil
}

// nativeLagrangeMultipliersPartial dispatches getLagrangeMultipliersPartial to
// the native fr.Element implementation for supported curves.
func nativeLagrangeMultipliersPartial(n uint64, c *mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, bool, error) {
	switch curve.GroupOrder.CurveID() {
	case mathlib.BLS12_381, mathlib.BLS12_381_GURVY, mathlib.BLS12_381_BBS, mathlib.BLS12_381_BBS_GURVY:
		r, err := getLagrangeMultipliersPartialNative[bls12381fr.Element, *bls12381fr.Element](n, c, curve)

		return r, true, err
	case mathlib.BN254:
		r, err := getLagrangeMultipliersPartialNative[bn254fr.Element, *bn254fr.Element](n, c, curve)

		return r, true, err
	}

	return nil, false, nil
}

// nativeInterpolate dispatches interpolate to the native fr.Element
// implementation for supported curves.
func nativeInterpolate(n uint64, vals []*mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, bool, error) {
	switch curve.GroupOrder.CurveID() {
	case mathlib.BLS12_381, mathlib.BLS12_381_GURVY, mathlib.BLS12_381_BBS, mathlib.BLS12_381_BBS_GURVY:
		r, err := interpolateNative[bls12381fr.Element, *bls12381fr.Element](n, vals, curve)

		return r, true, err
	case mathlib.BN254:
		r, err := interpolateNative[bn254fr.Element, *bn254fr.Element](n, vals, curve)

		return r, true, err
	}

	return nil, false, nil
}
