/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewTokensService(t *testing.T) {
	tests := []struct {
		name          string
		init          func() (logging.Logger, *setup.PublicParams, token2.IdentityDeserializer, error)
		check         func(pp *setup.PublicParams, ts *token2.TokensService) error
		wantErr       bool
		expectedError string
	}{
		{
			name: "publicParams cannot be nil",
			init: func() (logging.Logger, *setup.PublicParams, token2.IdentityDeserializer, error) {
				return nil, nil, nil, nil
			},
			wantErr:       true,
			expectedError: "publicParams cannot be nil",
		},
		{
			name: "identityDeserializer cannot be nil",
			init: func() (logging.Logger, *setup.PublicParams, token2.IdentityDeserializer, error) {
				pp, err := setup.Setup(32, nil, math.FP256BN_AMCL)
				if err != nil {
					return nil, nil, nil, err
				}
				return nil, pp, nil, nil
			},
			wantErr:       true,
			expectedError: "identityDeserializer cannot be nil",
		},
		{
			name: "success",
			init: func() (logging.Logger, *setup.PublicParams, token2.IdentityDeserializer, error) {
				pp, err := setup.Setup(32, []byte("issuer public key"), math.FP256BN_AMCL)
				if err != nil {
					return nil, nil, nil, err
				}
				return nil, pp, &mock.IdentityDeserializer{}, nil
			},
			check: func(pp *setup.PublicParams, ts *token2.TokensService) error {
				// check pp
				if ts.PublicParameters != pp {
					return errors.Errorf("public parameters not equal")
				}
				// check OutputTokenFormat
				outputTokenFormat, err := token2.ComputeTokenFormat(ts.PublicParameters, 32)
				if err != nil {
					return err
				}
				if ts.OutputTokenFormat != outputTokenFormat {
					return errors.Errorf("invalid token format [%s]", ts.OutputTokenFormat)
				}

				if len(ts.SupportedTokenFormats()) != 4 {
					return errors.Errorf("invalid number of supported token formats [%d]", len(ts.SupportedTokenFormats()))
				}
				dlog16, err1 := token2.ComputeTokenFormat(pp, 16)
				dlog32, err2 := token2.ComputeTokenFormat(pp, 32)
				ft16, err3 := v1.ComputeTokenFormat(16)
				ft32, err4 := v1.ComputeTokenFormat(32)
				if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
					return errors.Errorf("failed computing token format")
				}
				stf := collections.NewSet[token.Format](ts.SupportedTokenFormats()...)
				if !stf.Contains(dlog16) {
					return errors.Errorf("stf does not contain dlog16")
				}
				if !stf.Contains(dlog32) {
					return errors.Errorf("stf does not contain dlog32")
				}
				if !stf.Contains(ft16) {
					return errors.Errorf("stf does not contain ft16")
				}
				if !stf.Contains(ft32) {
					return errors.Errorf("stf does not contain ft32")
				}
				return nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, pp, deserializer, err := tt.init()
			assert.NoError(t, err)
			ts, err := token2.NewTokensService(logger, pp, deserializer)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
				assert.Nil(t, ts)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ts)
				assert.NoError(t, tt.check(pp, ts))
			}
		})
	}

}
