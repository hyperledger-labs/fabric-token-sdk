/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"runtime"
	"testing"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/benchmark"
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

func BenchmarkParallelKmIdentity(b *testing.B) {
	b.Run("BLS12_381_BBS_GURVY", func(b *testing.B) {
		b.ReportAllocs()

		keyManager, cleanup := setupKeyManager(b, "./testdata/bls12_381_bbs_gurvy/idemix", math.BLS12_381_BBS_GURVY)
		defer cleanup()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = keyManager.Identity(b.Context(), nil)
			}
		})
	})
}

func TestParallelBenchmarkIdemixKMIdentity(t *testing.T) {
	keyManager, cleanup := setupKeyManager(t, "./testdata/bls12_381_bbs_gurvy/idemix", math.BLS12_381_BBS_GURVY)
	defer cleanup()

	r := benchmark.RunBenchmark(
		runtime.NumCPU(),
		2*time.Minute,
		func() *KeyManager {
			return keyManager
		},
		func(km *KeyManager) {
			_, _ = keyManager.Identity(t.Context(), nil)
		},
	)
	r.Print()
}
