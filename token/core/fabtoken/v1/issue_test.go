/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIssueService(t *testing.T) {
	ppm := &mock.PublicParamsManager{}
	ws := &mock.WalletService{}
	des := &mock.Deserializer{}
	service := NewIssueService(ppm, ws, des)
	assert.NotNil(t, service)
	assert.Equal(t, ppm, service.PublicParamsManager)
	assert.Equal(t, ws, service.WalletService)
	assert.Equal(t, des, service.Deserializer)
}

func TestIssue(t *testing.T) {
	ctx := context.Background()
	issuer := driver.Identity("issuer")
	tokenType := token.Type("ABC")
	values := []uint64{100, 200}
	owners := [][]byte{[]byte("owner1"), []byte("owner2")}

	t.Run("Success", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)

		des.GetAuditInfoReturns([]byte("audit-info"), nil)

		action, metadata, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.NoError(t, err)
		assert.NotNil(t, action)
		assert.NotNil(t, metadata)
		assert.Equal(t, issuer, metadata.Issuer.Identity)
		assert.Equal(t, []byte("audit-info"), metadata.Issuer.AuditInfo)
		assert.Len(t, metadata.Outputs, 2)
	})

	t.Run("EmptyOwnerError", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		_, _, err := service.Issue(ctx, issuer, tokenType, values, [][]byte{[]byte("owner1"), nil}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all recipients should be defined")
	})

	t.Run("InvalidArgumentsError", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		_, _, err := service.Issue(ctx, nil, "", nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issuer identity, token type and values should be defined")
	})

	t.Run("RedeemNotSupported", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		opts := &driver.IssueOptions{
			TokensUpgradeRequest: &driver.TokenUpgradeRequest{},
		}
		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redeem during issue is not supported")
	})

	t.Run("GetAuditInfoError", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)

		des.GetAuditInfoReturns(nil, assert.AnError)

		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.Error(t, err)
	})

	t.Run("IssuerAuditInfoError", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)

		// Success for owners in loop (called twice)
		des.GetAuditInfoReturnsOnCall(0, []byte("audit1"), nil)
		des.GetAuditInfoReturnsOnCall(1, []byte("audit2"), nil)
		// Failure for issuer
		des.GetAuditInfoReturnsOnCall(2, nil, assert.AnError)

		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get audit info for issuer identity")
	})

	t.Run("PrecisionError", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(0) // Invalid precision
		ppm.PublicParametersReturns(pp)

		_, _, err := service.Issue(ctx, issuer, tokenType, values, owners, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to convert")
	})

	t.Run("IssueWithAttributes", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		ws := &mock.WalletService{}
		des := &mock.Deserializer{}
		service := NewIssueService(ppm, ws, des)

		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)
		des.GetAuditInfoReturns([]byte("audit"), nil)

		opts := &driver.IssueOptions{
			Attributes: map[interface{}]interface{}{"key": "value"},
		}
		action, _, err := service.Issue(ctx, issuer, tokenType, values, owners, opts)
		require.NoError(t, err)
		assert.NotNil(t, action)
	})
}

func TestVerifyIssue(t *testing.T) {
	service := NewIssueService(nil, nil, nil)
	err := service.VerifyIssue(context.Background(), nil, nil)
	require.NoError(t, err)
}

func TestDeserializeIssueAction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ppm := &mock.PublicParamsManager{}
		pp := &mock.PublicParameters{}
		pp.PrecisionReturns(64)
		ppm.PublicParametersReturns(pp)
		des := &mock.Deserializer{}
		des.GetAuditInfoReturns([]byte("audit"), nil)
		s := NewIssueService(ppm, nil, des)

		issuer := driver.Identity("issuer")
		action, _, err := s.Issue(context.Background(), issuer, "ABC", []uint64{100}, [][]byte{[]byte("owner")}, nil)
		require.NoError(t, err)

		raw, err := action.Serialize()
		require.NoError(t, err)

		service := NewIssueService(nil, nil, nil)
		deserialized, err := service.DeserializeIssueAction(raw)
		require.NoError(t, err)
		assert.NotNil(t, deserialized)
	})

	t.Run("Error", func(t *testing.T) {
		service := NewIssueService(nil, nil, nil)
		_, err := service.DeserializeIssueAction([]byte("invalid"))
		require.Error(t, err)
	})
}

// BenchmarkIssueServiceIssue benchmarks the Issue method of the IssueService.
func BenchmarkIssueServiceIssue(b *testing.B) {
	_, _, cases, err := benchmark2.GenerateCasesWithDefaults()
	require.NoError(b, err)

	for _, tc := range cases {
		b.Run(tc.Name, func(b *testing.B) {
			env, err := newBenchmarkIssueEnv(b.N, tc.BenchmarkCase)
			require.NoError(b, err)

			b.ResetTimer()
			i := 0
			for b.Loop() {
				e := env.Envs[i%len(env.Envs)]
				action, _, err := e.is.Issue(
					b.Context(),
					e.issuer,
					e.tokenType,
					e.values,
					e.owners,
					nil,
				)
				require.NoError(b, err)
				require.NotNil(b, action)
				i++
			}
		})
	}
}

// TestParallelBenchmarkIssueServiceIssue runs the issue benchmark in parallel.
func TestParallelBenchmarkIssueServiceIssue(t *testing.T) {
	_, _, cases, err := benchmark2.GenerateCasesWithDefaults()
	require.NoError(t, err)

	test := benchmark2.NewTest[*benchmarkIssueEnv](cases)
	test.RunBenchmark(t,
		func(c *benchmark2.Case) (*benchmarkIssueEnv, error) {
			return newBenchmarkIssueEnv(1, c)
		},
		func(ctx context.Context, env *benchmarkIssueEnv) error {
			action, _, err := env.Envs[0].is.Issue(
				ctx,
				env.Envs[0].issuer,
				env.Envs[0].tokenType,
				env.Envs[0].values,
				env.Envs[0].owners,
				nil,
			)
			if err != nil {
				return err
			}
			_, err = action.Serialize()
			return err
		},
	)
}

type issueEnv struct {
	is        *IssueService
	issuer    driver.Identity
	tokenType token.Type
	values    []uint64
	owners    [][]byte
}

func newIssueEnv(benchmarkCase *benchmark2.Case) (*issueEnv, error) {
	ppm := &mock.PublicParamsManager{}
	pp := &mock.PublicParameters{}
	pp.PrecisionReturns(64)
	ppm.PublicParametersReturns(pp)

	ws := &mock.WalletService{}
	des := &mock.Deserializer{}
	is := NewIssueService(ppm, ws, des)

	issuer := driver.Identity("issuer")
	tokenType := token.Type("ABC")
	values := make([]uint64, benchmarkCase.NumOutputs)
	owners := make([][]byte, benchmarkCase.NumOutputs)
	for i := range values {
		values[i] = uint64(i)*10 + 10
		owners[i] = []byte("owner")
	}

	des.GetAuditInfoReturns([]byte("audit"), nil)

	return &issueEnv{
		is:        is,
		issuer:    issuer,
		tokenType: tokenType,
		values:    values,
		owners:    owners,
	}, nil
}

type benchmarkIssueEnv struct {
	Envs []*issueEnv
}

func newBenchmarkIssueEnv(n int, benchmarkCase *benchmark2.Case) (*benchmarkIssueEnv, error) {
	envs := make([]*issueEnv, n)
	for i := range n {
		env, err := newIssueEnv(benchmarkCase)
		if err != nil {
			return nil, err
		}
		envs[i] = env
	}
	return &benchmarkIssueEnv{Envs: envs}, nil
}
