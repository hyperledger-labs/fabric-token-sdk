/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"testing"

	math "github.com/IBM/mathlib"
)

func BenchmarkKmIdentity(b *testing.B) {
	b.Run("FP256BN_AMCL", func(b *testing.B) {
		b.ReportAllocs()

		keyManager, cleanup := setupKeyManager(b, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
		defer cleanup()
		for b.Loop() {
			_, _ = keyManager.Identity(b.Context(), nil)
		}
	})

	b.Run("BLS12_381_BBS", func(b *testing.B) {
		b.ReportAllocs()

		// in this case, the backed uses GURVY directly
		keyManager, cleanup := setupKeyManager(b, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS)
		defer cleanup()
		for b.Loop() {
			_, _ = keyManager.Identity(b.Context(), nil)
		}
	})

	b.Run("BLS12_381_BBS_GURVY", func(b *testing.B) {
		b.ReportAllocs()

		keyManager, cleanup := setupKeyManager(b, "./testdata/bls12_381_bbs_gurvy/idemix", math.BLS12_381_BBS_GURVY)
		defer cleanup()
		for b.Loop() {
			_, _ = keyManager.Identity(b.Context(), nil)
		}
	})
}
