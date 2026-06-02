/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1_test

import (
	"context"
	"testing"

	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

// BenchmarkAuditorServiceCheck benchmarks the AuditorCheck method of the AuditorService.
func BenchmarkAuditorServiceCheck(b *testing.B) {
	_, _, cases, err := benchmark2.GenerateCasesWithDefaults()
	require.NoError(b, err)

	for _, tc := range cases {
		b.Run(tc.Name, func(b *testing.B) {
			env, err := newBenchmarkAuditEnv(b.N, tc.BenchmarkCase)
			require.NoError(b, err)

			b.ResetTimer()
			i := 0
			for b.Loop() {
				e := env.Envs[i%len(env.Envs)]
				err := e.as.AuditorCheck(
					b.Context(),
					e.request,
					e.metadata,
					e.anchor,
				)
				require.NoError(b, err)
				i++
			}
		})
	}
}

// TestParallelBenchmarkAuditorServiceCheck runs the auditor check benchmark in parallel.
func TestParallelBenchmarkAuditorServiceCheck(t *testing.T) {
	_, _, cases, err := benchmark2.GenerateCasesWithDefaults()
	require.NoError(t, err)

	test := benchmark2.NewTest[*benchmarkAuditEnv](cases)
	test.RunBenchmark(t,
		func(c *benchmark2.Case) (*benchmarkAuditEnv, error) {
			return newBenchmarkAuditEnv(1, c)
		},
		func(ctx context.Context, env *benchmarkAuditEnv) error {
			e := env.Envs[0]

			return e.as.AuditorCheck(
				ctx,
				e.request,
				e.metadata,
				e.anchor,
			)
		},
	)
}

type auditEnv struct {
	as       *v1.AuditorService
	request  *driver.TokenRequest
	metadata *driver.TokenRequestMetadata
	anchor   driver.TokenRequestAnchor
}

type benchmarkAuditEnv struct {
	Envs []*auditEnv
}

func newBenchmarkAuditEnv(n int, benchmarkCase *benchmark2.Case) (*benchmarkAuditEnv, error) {
	envs := make([]*auditEnv, n)
	for i := range n {
		env, err := newAuditEnv(benchmarkCase)
		if err != nil {
			return nil, err
		}
		envs[i] = env
	}

	return &benchmarkAuditEnv{Envs: envs}, nil
}

func newAuditEnv(benchmarkCase *benchmark2.Case) (*auditEnv, error) {
	as := v1.NewAuditorService()

	// In fabtoken, AuditorCheck is a no-op, but we still populate the structures
	// to match the expected usage pattern and ensure deserialization works if ever added.

	issueAction := &actions.IssueAction{
		Issuer: []byte("issuer"),
	}
	for i := range benchmarkCase.NumOutputs {
		issueAction.Outputs = append(issueAction.Outputs, &actions.Output{
			Owner:    []byte("owner"),
			Type:     "ABC",
			Quantity: token.NewQuantityFromUInt64(uint64(i)*10 + 10).Hex(),
		})
	}
	rawAction, err := issueAction.Serialize()
	if err != nil {
		return nil, err
	}

	request := &driver.TokenRequest{
		Issues: [][]byte{rawAction},
	}

	metadata := &driver.TokenRequestMetadata{
		Issues: []*driver.IssueMetadata{
			{
				Issuer: driver.AuditableIdentity{
					Identity:  []byte("issuer"),
					AuditInfo: []byte("audit-info"),
				},
				Outputs: []*driver.IssueOutputMetadata{
					{
						OutputMetadata: []byte("token-metadata"),
						Receivers: []*driver.AuditableIdentity{
							{
								Identity:  []byte("owner"),
								AuditInfo: []byte("audit-info"),
							},
						},
					},
				},
			},
		},
	}

	return &auditEnv{
		as:       as,
		request:  request,
		metadata: metadata,
		anchor:   "benchmark-anchor",
	}, nil
}
