/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator_test

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/benchmark"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/testdata"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	tk "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

var (
	testUseCase = &benchmark2.Case{
		Bits:       32,
		CurveID:    math.BLS12_381_BBS_GURVY,
		NumInputs:  2,
		NumOutputs: 2,
	}
)

type actionType int

const (
	TransferAction actionType = iota
	RedeemAction
	IssueAction
)

//go:embed testdata
var testDataFS embed.FS

func TestValidator(t *testing.T) {
	t.Run("Validator is called correctly with a non-anonymous issue action", func(t *testing.T) {
		testVerifyNoErrorOnAction(t, IssueAction)
	})
	t.Run("validator is called correctly with a transfer action", func(t *testing.T) {
		testVerifyNoErrorOnAction(t, TransferAction)
	})
	t.Run("validator is called correctly with a redeem action", func(t *testing.T) {
		testVerifyNoErrorOnAction(t, RedeemAction)
	})
	t.Run("engine is called correctly with atomic swap", func(t *testing.T) {
		configurations, err := benchmark.NewSetupConfigurations("./../testdata", []uint64{testUseCase.Bits}, []math.CurveID{testUseCase.CurveID})
		require.NoError(t, err)
		env, err := testdata.NewEnv(testUseCase, configurations)
		require.NoError(t, err)

		raw, err := env.TRWithSwap.Bytes()
		require.NoError(t, err)

		actions, _, err := env.Engine.VerifyTokenRequestFromRaw(t.Context(), nil, "2", raw)
		require.NoError(t, err)
		require.Len(t, actions, 1)
	})
	t.Run("when the sender's signature is not valid: wrong txID", func(t *testing.T) {
		configurations, err := benchmark.NewSetupConfigurations("./../testdata", []uint64{testUseCase.Bits}, []math.CurveID{testUseCase.CurveID})
		require.NoError(t, err)
		env, err := testdata.NewEnv(testUseCase, configurations)
		require.NoError(t, err)

		request := &driver.TokenRequest{Issues: env.TRWithSwap.Issues, Transfers: env.TRWithSwap.Transfers}
		raw, err := request.MarshalToMessageToSign([]byte("3"))
		require.NoError(t, err)

		signatures, err := env.Sender.SignTokenActions(raw)
		require.NoError(t, err)
		env.TRWithSwap.Signatures[1] = signatures[0]

		raw, err = env.TRWithSwap.Bytes()
		require.NoError(t, err)

		_, _, err = env.Engine.VerifyTokenRequestFromRaw(t.Context(), nil, "2", raw)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed signature verification")
	})
}

func TestParallelBenchmarkValidatorTransfer(t *testing.T) {
	bits, curves, cases, err := benchmark2.GenerateCasesWithDefaults()
	require.NoError(t, err)
	configurations, err := benchmark.NewSetupConfigurations("./../testdata", bits, curves)
	require.NoError(t, err)

	test := benchmark2.NewTest[*testdata.Env](cases)
	test.RunBenchmark(t,
		func(c *benchmark2.Case) (*testdata.Env, error) {
			return testdata.NewEnv(c, configurations)
		},
		func(ctx context.Context, env *testdata.Env) error {
			_, _, err := env.Engine.VerifyTokenRequestFromRaw(ctx, nil, "1", env.TRWithTransferRaw)
			return err
		},
	)
}

func TestRegression(t *testing.T) {
	testRegression(t, "testdata/32-BLS12_381_BBS_GURVY")
	testRegression(t, "testdata/64-BLS12_381_BBS_GURVY")
	testRegression(t, "testdata/32-BN254")
	testRegression(t, "testdata/64-BN254")
}

func testRegression(t *testing.T, rootDir string) {
	t.Helper()
	paramsData, err := testDataFS.ReadFile(filepath.Join(rootDir, "params.txt"))
	require.NoError(t, err)

	ppRaw, err := base64.StdEncoding.DecodeString(string(paramsData))
	require.NoError(t, err)

	_, tokenValidator, err := tokenServicesFactory(ppRaw)
	require.NoError(t, err)

	var tokenData struct {
		ReqRaw []byte `json:"req_raw"`
		TXID   string `json:"txid"`
	}
	for i := range 64 {
		jsonData, err := testDataFS.ReadFile(filepath.Join(rootDir, "transfers", fmt.Sprintf("output.%d.json", i)))
		require.NoError(t, err)
		err = json.Unmarshal(jsonData, &tokenData)
		require.NoError(t, err)
		_, _, err = tokenValidator.UnmarshallAndVerifyWithMetadata(
			context.Background(),
			&fakeLedger{},
			token.RequestAnchor(tokenData.TXID),
			tokenData.ReqRaw,
		)
		require.NoError(t, err)
	}
}

func testVerifyNoErrorOnAction(t *testing.T, actionType actionType) {
	t.Helper()
	configurations, err := benchmark.NewSetupConfigurations("./../testdata", []uint64{testUseCase.Bits}, []math.CurveID{testUseCase.CurveID})
	require.NoError(t, err)
	env, err := testdata.NewEnv(testUseCase, configurations)
	require.NoError(t, err)

	var raw []byte
	switch actionType {
	case TransferAction:
		raw, err = env.TRWithTransfer.Bytes()
	case IssueAction:
		raw, err = env.TRWithIssue.Bytes()
	case RedeemAction:
		raw, err = env.TRWithRedeem.Bytes()
	}
	require.NoError(t, err)
	actions, _, err := env.Engine.VerifyTokenRequestFromRaw(t.Context(), nil, "1", raw)
	require.NoError(t, err)
	require.Len(t, actions, 1)
}

func tokenServicesFactory(bytes []byte) (tcc.PublicParameters, tcc.Validator, error) {
	is := core.NewPPManagerFactoryService(fabtoken.NewPPMFactory(), dlog.NewPPMFactory())

	ppm, err := is.PublicParametersFromBytes(bytes)
	if err != nil {
		return nil, nil, err
	}
	v, err := is.DefaultValidator(ppm)
	if err != nil {
		return nil, nil, err
	}
	return ppm, token.NewValidator(v), nil
}

type fakeLedger struct{}

func (*fakeLedger) GetState(id tk.ID) ([]byte, error) {
	panic("ciao")
}
