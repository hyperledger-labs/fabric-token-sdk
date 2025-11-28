/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	v1token "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestTransferService_VerifyTransfer(t *testing.T) {
	tests := []struct {
		name     string
		TestCase func() (*TransferService, driver.TransferAction, []*driver.TransferOutputMetadata)
		wantErr  string
	}{
		{
			name: "nil action",
			TestCase: func() (*TransferService, driver.TransferAction, []*driver.TransferOutputMetadata) {
				service := &TransferService{}
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

func TestA(t *testing.T) {
	logger := logging.MustGetLogger()
	pp, err := v1.Setup(32, nil, math.BLS12_377_GURVY)
	require.NoError(t, err)
	ppm, err := common.NewPublicParamsManagerFromParams(pp)
	require.NoError(t, err)
	deserializer, err := driver2.NewDeserializer(pp)
	require.NoError(t, err)
	tokensService, err := v1token.NewTokensService(logger, ppm, deserializer)
	require.NoError(t, err)

	ids := []*token.ID{}
	outputs := []*token.Token{}
	ts := NewTransferService(
		logger,
		ppm,
		nil,
		nil,
		deserializer,
		NewMetrics(&disabled.Provider{}),
		noop.NewTracerProvider(),
		tokensService,
	)
	action, _, err := ts.Transfer(
		t.Context(),
		"an_anchor",
		nil,
		ids,
		outputs,
		nil,
	)
	require.NoError(t, err)
	assert.NotNil(t, action)
}
