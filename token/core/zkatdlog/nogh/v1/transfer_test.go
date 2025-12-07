/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1_test

import (
	"strconv"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/benchmark"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/mock"
	v1token "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

var logger = logging.MustGetLogger()

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

func BenchmarkTransferServiceTransfer(b *testing.B) {
	bits, err := benchmark2.Bits(32, 64)
	require.NoError(b, err)
	curves := benchmark2.Curves(math.BN254, math.BLS12_381_BBS_GURVY, math2.BLS12_381_BBS_GURVY_FAST_RNG)
	inputs, err := benchmark2.NumInputs(1, 2, 3)
	require.NoError(b, err)
	outputs, err := benchmark2.NumOutputs(1, 2, 3)
	require.NoError(b, err)
	testCases := benchmark2.GenerateCases(bits, curves, inputs, outputs, []int{1})

	for _, tc := range testCases {
		b.Run(tc.Name, func(b *testing.B) {
			env, err := newBenchmarkTransferEnv(b.N, tc.BenchmarkCase, nil)
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

func TestParallelBenchmarkTransferServiceTransfer(t *testing.T) {
	bits, curves, cases := benchmark2.GenerateCasesWithDefaults(t)
	configurations, err := benchmark.NewSetupConfigurations("./testdata", bits, curves)
	require.NoError(t, err)

	test := benchmark2.NewTest[*benchmarkTransferEnv](cases)
	test.RunBenchmark(t,
		func(c *benchmark2.Case) (*benchmarkTransferEnv, error) {
			return newBenchmarkTransferEnv(1, c, configurations)
		},
		func(env *benchmarkTransferEnv) error {
			action, _, err := env.Envs[0].ts.Transfer(
				t.Context(),
				"an_anchor",
				nil,
				env.Envs[0].ids,
				env.Envs[0].outputs,
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

type transferEnv struct {
	ts      *v1.TransferService
	outputs []*token.Token
	ids     []*token.ID
}

func newTransferEnv(benchmarkCase *benchmark2.Case, configurations *benchmark.SetupConfigurations) (*transferEnv, error) {
	pp, err := configurations.GetPublicParams(benchmarkCase.Bits, benchmarkCase.CurveID)
	if err != nil {
		return nil, err
	}
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

func newBenchmarkTransferEnv(n int, benchmarkCase *benchmark2.Case, configurations *benchmark.SetupConfigurations) (*benchmarkTransferEnv, error) {
	envs := make([]*transferEnv, n)
	for i := 0; i < n; i++ {
		env, err := newTransferEnv(benchmarkCase, configurations)
		if err != nil {
			return nil, err
		}
		envs[i] = env
	}
	return &benchmarkTransferEnv{Envs: envs}, nil
}
