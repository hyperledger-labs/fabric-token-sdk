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
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/stretchr/testify/require"
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

func TestParallelBenchmarkIdemixKMIdentity(t *testing.T) {
	keyManager, cleanup := setupKeyManager(t, "./testdata/bls12_381_bbs_gurvy/idemix", math.BLS12_381_BBS_GURVY)
	defer cleanup()

	workers, err := benchmark2.Workers(runtime.NumCPU())
	require.NoError(t, err)

	r := benchmark2.RunBenchmark(
		benchmark2.NewConfig(workers[0],
			benchmark2.Duration(),
			3*time.Second),
		func() *KeyManager {
			return keyManager
		},
		func(km *KeyManager) error {
			_, err := keyManager.Identity(t.Context(), nil)
			return err
		},
	)
	r.Print()
}
