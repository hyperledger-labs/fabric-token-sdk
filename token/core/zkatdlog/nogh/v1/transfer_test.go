/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1_test

import (
	"os"
	"runtime"
	"strconv"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/benchmark"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/mock"
	v1setup "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	v1token "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestTransferService_VerifyTransfer(t *testing.T) {
	tests := []struct {
		name     string
		TestCase func() (*v1.TransferService, driver.TransferAction, []*driver.TransferOutputMetadata)
		wantErr  string
	}{
		{
			name: "nil action",
			TestCase: func() (*v1.TransferService, driver.TransferAction, []*driver.TransferOutputMetadata) {
				service := &v1.TransferService{}
				return service, nil, nil
			},
			wantErr: "nil action",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, action, meta := tt.TestCase()
			err := service.VerifyTransfer(t.Context(), action, meta)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func BenchmarkTransfer(b *testing.B) {
	bits, err := benchmark.Bits(32, 64)
	require.NoError(b, err)
	curves := benchmark.Curves(math.BN254, math.BLS12_381_BBS_GURVY, math2.BLS12_381_BBS_GURVY_FAST_RNG)
	inputs, err := benchmark.NumInputs(1, 2, 3)
	require.NoError(b, err)
	outputs, err := benchmark.NumOutputs(1, 2, 3)
	require.NoError(b, err)
	testCases := benchmark.GenerateCases(bits, curves, inputs, outputs, []int{1})

	for _, tc := range testCases {
		b.Run(tc.Name, func(b *testing.B) {
			env, err := newBenchmarkTransferEnv(b.N, tc.BenchmarkCase)
			require.NoError(b, err)

			// Optional: Reset timer if you had expensive setup code above
			b.ResetTimer()

			i := 0
			for b.Loop() {
				action, _, err := env.Envs[i].ts.Transfer(
					b.Context(),
					"an_anchor",
					nil,
					env.Envs[i].ids,
					env.Envs[i].outputs,
					nil,
				)
				require.NoError(b, err)
				assert.NotNil(b, action)
				i++
			}
		})
	}
}

func TestBenchmarkTransferParallel(t *testing.T) {
	bits, err := benchmark.Bits(32)
	require.NoError(t, err)
	curves := benchmark.Curves(math.BN254)
	inputs, err := benchmark.NumInputs(2)
	require.NoError(t, err)
	outputs, err := benchmark.NumOutputs(2)
	require.NoError(t, err)
	workers, err := benchmark.Workers(runtime.NumCPU())
	require.NoError(t, err)
	testCases := benchmark.GenerateCases(bits, curves, inputs, outputs, workers)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			r := benchmark.RunBenchmark(
				tc.BenchmarkCase.Workers,
				benchmark.Duration(),
				func() *benchmarkTransferEnv {
					env, err := newBenchmarkTransferEnv(1, tc.BenchmarkCase)
					require.NoError(t, err)
					return env
				},
				func(env *benchmarkTransferEnv) {
					action, _, err := env.Envs[0].ts.Transfer(
						t.Context(),
						"an_anchor",
						nil,
						env.Envs[0].ids,
						env.Envs[0].outputs,
						nil,
					)
					require.NoError(t, err)
					assert.NotNil(t, action)
					raw, err := action.Serialize()
					require.NoError(t, err)
					require.NotEmpty(t, raw)
				},
			)
			r.Print()
		})
	}
}

type transferEnv struct {
	ts      *v1.TransferService
	outputs []*token.Token
	ids     []*token.ID
}

func newTransferEnv(benchmarkCase *benchmark.Case) (*transferEnv, error) {
	logger := logging.MustGetLogger()

	var ipk []byte
	var err error
	switch benchmarkCase.CurveID {
	case math.BN254:
		ipk, err = os.ReadFile("./validator/testdata/bn254/idemix/msp/IssuerPublicKey")
		if err != nil {
			return nil, err
		}
	case math.BLS12_381_BBS_GURVY:
		fallthrough
	case math2.BLS12_381_BBS_GURVY_FAST_RNG:
		ipk, err = os.ReadFile("./validator/testdata/bls12_381_bbs/idemix/msp/IssuerPublicKey")
		if err != nil {
			return nil, err
		}
	}

	pp, err := v1setup.Setup(benchmarkCase.Bits, ipk, benchmarkCase.CurveID)
	if err != nil {
		return nil, err
	}
	pp.AddIssuer([]byte("an_issuer"))
	ppm, err := common.NewPublicParamsManagerFromParams(pp)
	if err != nil {
		return nil, err
	}
	deserializer, err := driver2.NewDeserializer(pp)
	if err != nil {
		return nil, err
	}
	tokensService, err := v1token.NewTokensService(logger, ppm, deserializer)
	if err != nil {
		return nil, err
	}

	ownerID, err := identity.WrapWithType(idemix.IdentityType, []byte("alice"))
	if err != nil {
		return nil, err
	}
	outputs := make([]*token.Token, benchmarkCase.NumOutputs)
	for i := 0; i < benchmarkCase.NumOutputs; i++ {
		outputs[i] = &token.Token{
			Owner:    ownerID,
			Quantity: token.NewQuantityFromUInt64(uint64(i*10 + 10)).Hex(),
			Type:     "ABC",
		}
	}

	// prepare inputs
	numInputs := benchmarkCase.NumInputs
	ids := make([]*token.ID, numInputs)
	values := make([]uint64, numInputs)
	for i := 0; i < numInputs; i++ {
		values[i] = uint64(i*10 + 10)
	}
	baseTokens, metadata, err := v1token.GetTokensWithWitness(values, "ABC", pp.PedersenGenerators, math.Curves[pp.Curve])
	if err != nil {
		return nil, err
	}
	var loadedTokens []v1.LoadedToken
	tokenFormat, err := v1token.SupportedTokenFormat(pp, benchmarkCase.Bits)
	if err != nil {
		return nil, err
	}

	for i, tok := range baseTokens {
		ownerID, err := identity.WrapWithType(idemix.IdentityType, []byte("alice"))
		if err != nil {
			return nil, err
		}
		v1Token := &v1token.Token{
			Owner: ownerID,
			Data:  tok,
		}
		tokenRaw, err := v1Token.Serialize()
		if err != nil {
			return nil, err
		}

		// metadata
		mdRaw, err := metadata[i].Serialize()
		if err != nil {
			return nil, err
		}

		loadedTokens = append(loadedTokens, v1.LoadedToken{
			TokenFormat: tokenFormat,
			Token:       tokenRaw,
			Metadata:    mdRaw,
		})
		ids[i] = &token.ID{TxId: strconv.Itoa(i)}
	}
	tokenLoader := &mock.TokenLoader{}
	tokenLoader.LoadTokensReturns(loadedTokens, nil)

	auditInfoProvider := &mock2.AuditInfoProvider{}
	auditInfoProvider.GetAuditInfoReturns([]byte("auditInfo"), nil)

	ts := v1.NewTransferService(
		logger,
		ppm,
		auditInfoProvider,
		tokenLoader,
		deserializer,
		v1.NewMetrics(&disabled.Provider{}),
		noop.NewTracerProvider(),
		tokensService,
	)
	return &transferEnv{
		ts:      ts,
		outputs: outputs,
		ids:     ids,
	}, nil
}

type benchmarkTransferEnv struct {
	Envs []*transferEnv
}

func newBenchmarkTransferEnv(n int, benchmarkCase *benchmark.Case) (*benchmarkTransferEnv, error) {
	envs := make([]*transferEnv, n)
	for i := 0; i < n; i++ {
		env, err := newTransferEnv(benchmarkCase)
		if err != nil {
			return nil, err
		}
		envs[i] = env
	}
	return &benchmarkTransferEnv{Envs: envs}, nil
}
