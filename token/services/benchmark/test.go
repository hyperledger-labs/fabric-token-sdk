/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"math/rand"
	"runtime"
	"testing"
	"time"

	math "github.com/IBM/mathlib"
	fscprofile "github.com/hyperledger-labs/fabric-smart-client/node/start/profile"
	"github.com/stretchr/testify/require"
)

// GenerateCasesWithDefaults returns all combinations of Case created from the value of the flags:
// bits, curves, num_inputs, num_outputs, workers.
// It uses the following predefined values:
// - bits: 32
// - curves: BN254
// - num_inputs: 2
// - num_outputs: 2
// - workers: NumCPU
func GenerateCasesWithDefaults(tb testing.TB) ([]uint64, []math.CurveID, []TestCase) {
	tb.Helper()
	bits, err := Bits(32)
	require.NoError(tb, err)
	curves := Curves(math.BN254)
	inputs, err := NumInputs(2)
	require.NoError(tb, err)
	outputs, err := NumOutputs(2)
	require.NoError(tb, err)
	workers, err := Workers(runtime.NumCPU())
	require.NoError(tb, err)
	return bits, curves, GenerateCases(bits, curves, inputs, outputs, workers)
}

type Test[T any] struct {
	TestCases []TestCase
}

func NewTest[T any](testCases []TestCase) *Test[T] {
	return &Test[T]{TestCases: testCases}
}

func (test *Test[T]) GoBenchmark(b *testing.B, newEnv func(*Case) (T, error), work func(env T) error) {
	b.Helper()
	if ProfileEnabled() {
		p, err := fscprofile.New(fscprofile.WithAll(), fscprofile.WithPath("./profile"))
		require.NoError(b, err)
		require.NoError(b, p.Start())
		defer p.Stop()
	}

	for _, tc := range test.TestCases {
		b.Run(tc.Name, func(b *testing.B) {
			n := SetupSamples()
			envs := make([]T, 0, n)
			if n == 0 {
				n = uint(b.N)
			}
			if n > 0 {
				for range n {
					e, err := newEnv(tc.BenchmarkCase)
					require.NoError(b, err)
					envs = append(envs, e)
				}
			}

			for b.Loop() {
				require.NoError(b, work(envs[rand.Intn(int(n))]))
			}
		})
	}
}

func (test *Test[T]) GoBenchmarkParallel(b *testing.B, newEnv func(*Case) (T, error), work func(env T) error) {
	b.Helper()
	if ProfileEnabled() {
		p, err := fscprofile.New(fscprofile.WithAll(), fscprofile.WithPath("./profile"))
		require.NoError(b, err)
		require.NoError(b, p.Start())
		defer p.Stop()
	}

	for _, tc := range test.TestCases {
		n := SetupSamples()
		envs := make([]T, 0, n)
		if n == 0 {
			n = uint(b.N)
		}
		if n > 0 {
			for range n {
				e, err := newEnv(tc.BenchmarkCase)
				require.NoError(b, err)
				envs = append(envs, e)
			}
		}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				require.NoError(b, work(envs[rand.Intn(int(n))]))
			}
		})
	}
}

func (test *Test[T]) RunBenchmark(t *testing.T, newEnv func(*Case) (T, error), work func(env T) error) {
	t.Helper()
	if ProfileEnabled() {
		p, err := fscprofile.New(fscprofile.WithAll(), fscprofile.WithPath("./profile"))
		require.NoError(t, err)
		require.NoError(t, p.Start())
		defer p.Stop()
	}

	for _, tc := range test.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			n := SetupSamples()
			envs := make([]T, 0, n)
			if n > 0 {
				for range n {
					e, err := newEnv(tc.BenchmarkCase)
					require.NoError(t, err)
					envs = append(envs, e)
				}
			}

			r := RunBenchmark(
				NewConfig(
					tc.BenchmarkCase.Workers,
					Duration(),
					3*time.Second),
				func() T {
					if n > 0 {
						return envs[rand.Intn(int(n))]
					}
					e, err := newEnv(tc.BenchmarkCase)
					require.NoError(t, err)
					return e
				},
				func(env T) error {
					return work(env)
				},
			)
			r.Print()
		})
	}
}
