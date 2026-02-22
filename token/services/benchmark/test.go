/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"context"
	"math/rand"
	"runtime"
	"testing"
	"time"

	math "github.com/IBM/mathlib"
	fscprofile "github.com/hyperledger-labs/fabric-smart-client/node/start/profile"
	"github.com/stretchr/testify/require"
)

// GenerateCasesWithDefaults returns all combinations of TestCase created from the
// value of the flags: bits, curves, num_inputs, num_outputs, workers.
// The concrete defaults used are: bits=32, curves=math.BN254, num_inputs=2,
// num_outputs=2 and workers=runtime.NumCPU().
// It is a helper for tests and benchmarks and calls tb.Helper().
func GenerateCasesWithDefaults() ([]uint64, []math.CurveID, []TestCase, error) {
	bits, err := Bits(32)
	if err != nil {
		return nil, nil, nil, err
	}
	curves := Curves(math.BN254)
	inputs, err := NumInputs(2)
	if err != nil {
		return nil, nil, nil, err
	}
	outputs, err := NumOutputs(2)
	if err != nil {
		return nil, nil, nil, err
	}
	workers, err := Workers(runtime.NumCPU())
	if err != nil {
		return nil, nil, nil, err
	}

	return bits, curves, GenerateCases(bits, curves, inputs, outputs, workers), nil
}

// Test groups a set of benchmark TestCase values. The type is generic and can
// be used with any environment value produced by the provided newEnv
// constructor when running the benchmark functions on Test.
type Test[T any] struct {
	TestCases []TestCase
}

// NewTest constructs and returns a pointer to a Test initialized with the
// provided testCases.
func NewTest[T any](testCases []TestCase) *Test[T] {
	return &Test[T]{TestCases: testCases}
}

// GoBenchmark runs the provided work function as standard go benchmarks using
// *testing.B. For each TestCase in the Test, it uses newEnv to create an
// environment value of type T and repeatedly invokes work on randomly chosen
// environments. If profiling is enabled, a profile is started and stopped
// around the benchmark. The function calls b.Helper() internally.
func (test *Test[T]) GoBenchmark(b *testing.B, newEnv func(*Case) (T, error), work func(ctx context.Context, env T) error) {
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
				n = uint(b.N) // #nosec G115
			}
			if n > 0 {
				for range n {
					e, err := newEnv(tc.BenchmarkCase)
					require.NoError(b, err)
					envs = append(envs, e)
				}
			}

			for b.Loop() {
				require.NoError(b, work(b.Context(), envs[rand.Intn(int(n))])) // #nosec G115
			}
		})
	}
}

// GoBenchmarkParallel runs the provided work function in parallel using
// b.RunParallel. For each TestCase it prepares a set of environments via
// newEnv and then executes work concurrently across goroutines managed by the
// testing package.
// Profiling is started if enabled.
// The method calls b.Helper().
func (test *Test[T]) GoBenchmarkParallel(b *testing.B, newEnv func(*Case) (T, error), work func(ctx context.Context, env T) error) {
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
			n = uint(b.N) // #nosec G115
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
				require.NoError(b, work(b.Context(), envs[rand.Intn(int(n))])) // #nosec G115
			}
		})
	}
}

// RunBenchmark runs the provided work function using the RunBenchmark harness
// (which returns a result that is printed). For each TestCase it prepares an
// environment constructor that either selects from prepared samples or creates
// a fresh environment via newEnv. Profiling is started if enabled. The
// function calls t.Helper().
func (test *Test[T]) RunBenchmark(t *testing.T, newEnv func(*Case) (T, error), work func(ctx context.Context, env T) error) {
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
						return envs[rand.Intn(int(n))] // #nosec G115
					}
					e, err := newEnv(tc.BenchmarkCase)
					require.NoError(t, err)

					return e
				},
				func(env T) error {
					return work(t.Context(), env)
				},
			)
			r.Print()
		})
	}
}
