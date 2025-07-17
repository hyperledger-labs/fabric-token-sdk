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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/comm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewTokensService(t *testing.T) {
	tests := []struct {
		name          string
		init          func() (*setup.PublicParams, token2.IdentityDeserializer, error)
		check         func(pp *setup.PublicParams, ts *token2.TokensService) error
		wantErr       bool
		expectedError string
	}{
		{
			name: "publicParams cannot be nil",
			init: func() (*setup.PublicParams, token2.IdentityDeserializer, error) {
				return nil, nil, nil
			},
			wantErr:       true,
			expectedError: "publicParams cannot be nil",
		},
		{
			name: "identityDeserializer cannot be nil",
			init: func() (*setup.PublicParams, token2.IdentityDeserializer, error) {
				pp, err := setup.Setup(32, nil, math.FP256BN_AMCL)
				if err != nil {
					return nil, nil, err
				}
				return pp, nil, nil
			},
			wantErr:       true,
			expectedError: "identityDeserializer cannot be nil",
		},
		{
			name: "success",
			init: func() (*setup.PublicParams, token2.IdentityDeserializer, error) {
				pp, err := setup.Setup(32, []byte("issuer public key"), math.FP256BN_AMCL)
				if err != nil {
					return nil, nil, err
				}
				return pp, &mock.IdentityDeserializer{}, nil
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
	logger := logging.MustGetLogger()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp, deserializer, err := tt.init()
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

func TestTokensService_Recipients(t *testing.T) {
	pp, err := setup.Setup(32, []byte("issuer public key"), math.FP256BN_AMCL)
	assert.NoError(t, err)

	tests := []struct {
		name               string
		inputs             func() (*token2.TokensService, driver.TokenOutput, error)
		wantErr            bool
		expectedError      string
		expectedIdentities []driver.Identity
	}{
		{
			name: "failed to deserialize token",
			inputs: func() (*token2.TokensService, driver.TokenOutput, error) {
				ts, err := token2.NewTokensService(nil, pp, &mock.IdentityDeserializer{})
				if err != nil {
					return nil, nil, err
				}
				return ts, nil, nil
			},
			wantErr:       true,
			expectedError: "failed to deserialize token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name: "failed to deserialize token 2",
			inputs: func() (*token2.TokensService, driver.TokenOutput, error) {
				ts, err := token2.NewTokensService(nil, pp, &mock.IdentityDeserializer{})
				if err != nil {
					return nil, nil, err
				}
				return ts, []byte{}, nil
			},
			wantErr:       true,
			expectedError: "failed to deserialize token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name: "failed to deserialize token 3",
			inputs: func() (*token2.TokensService, driver.TokenOutput, error) {
				ts, err := token2.NewTokensService(nil, pp, &mock.IdentityDeserializer{})
				if err != nil {
					return nil, nil, err
				}
				return ts, []byte{0, 1, 2, 3}, nil
			},
			wantErr:       true,
			expectedError: "failed to deserialize token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} TypedToken @2",
		},
		{
			name: "failed to deserialize token 4",
			inputs: func() (*token2.TokensService, driver.TokenOutput, error) {
				id := &mock.IdentityDeserializer{}
				id.RecipientsReturns(nil, nil)
				ts, err := token2.NewTokensService(nil, pp, id)
				if err != nil {
					return nil, nil, err
				}
				raw, err := comm.WrapTokenWithType([]byte{0, 1, 2, 3})
				if err != nil {
					return nil, nil, err
				}
				return ts, driver.TokenOutput(raw), nil
			},
			wantErr:       true,
			expectedError: "failed to deserialize token: failed unmarshalling token: proto: cannot parse invalid wire-format data",
		},
		{
			name: "failed identity deserialize",
			inputs: func() (*token2.TokensService, driver.TokenOutput, error) {
				id := &mock.IdentityDeserializer{}
				id.RecipientsReturns(nil, errors.New("pineapple"))
				ts, err := token2.NewTokensService(nil, pp, id)
				if err != nil {
					return nil, nil, err
				}
				tok := &token2.Token{}
				raw, err := tok.Serialize()
				if err != nil {
					return nil, nil, err
				}
				return ts, raw, nil
			},
			wantErr:       true,
			expectedError: "failed to get recipients: pineapple",
		},
		{
			name: "success",
			inputs: func() (*token2.TokensService, driver.TokenOutput, error) {
				id := &mock.IdentityDeserializer{}
				id.RecipientsReturns([]driver.Identity{driver.Identity("alice")}, nil)
				ts, err := token2.NewTokensService(nil, pp, id)
				if err != nil {
					return nil, nil, err
				}
				tok := &token2.Token{}
				raw, err := tok.Serialize()
				if err != nil {
					return nil, nil, err
				}
				return ts, raw, nil
			},
			wantErr:            false,
			expectedIdentities: []driver.Identity{driver.Identity("alice")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, outputs, err := tt.inputs()
			assert.NoError(t, err)
			identities, err := ts.Recipients(outputs)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
				assert.Nil(t, identities)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ts)
				assert.Equal(t, tt.expectedIdentities, identities)
			}
		})
	}
}

func TestTokensService_Deobfuscate(t *testing.T) {
	pp, err := setup.Setup(32, []byte("issuer public key"), math.FP256BN_AMCL)
	assert.NoError(t, err)

	tests := []struct {
		name          string
		inputs        func() (*token2.TokensService, driver.TokenOutput, driver.TokenOutputMetadata, error)
		wantErr       bool
		expectedError string
	}{
		{
			name: "failed to deserialize token",
			inputs: func() (*token2.TokensService, driver.TokenOutput, driver.TokenOutputMetadata, error) {
				ts, err := token2.NewTokensService(nil, pp, &mock.IdentityDeserializer{})
				if err != nil {
					return nil, nil, nil, err
				}
				return ts, nil, nil, nil
			},
			wantErr:       true,
			expectedError: "failed to deobfuscate: failed to deobfuscate comm token: failed to deobfuscate token: failed getting token output: failed to deserialize token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated\nfailed to deobfuscate fabtoken token: failed unmarshalling token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name: "failed to deserialize token 2",
			inputs: func() (*token2.TokensService, driver.TokenOutput, driver.TokenOutputMetadata, error) {
				ts, err := token2.NewTokensService(nil, pp, &mock.IdentityDeserializer{})
				if err != nil {
					return nil, nil, nil, err
				}
				return ts, []byte{}, nil, nil
			},
			wantErr:       true,
			expectedError: "failed to deobfuscate: failed to deobfuscate comm token: failed to deobfuscate token: failed getting token output: failed to deserialize token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated\nfailed to deobfuscate fabtoken token: failed unmarshalling token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name: "failed to deserialize token 3",
			inputs: func() (*token2.TokensService, driver.TokenOutput, driver.TokenOutputMetadata, error) {
				ts, err := token2.NewTokensService(nil, pp, &mock.IdentityDeserializer{})
				if err != nil {
					return nil, nil, nil, err
				}
				return ts, []byte{0, 1, 2, 3}, nil, nil
			},
			wantErr:       true,
			expectedError: "failed to deobfuscate: failed to deobfuscate comm token: failed to deobfuscate token: failed getting token output: failed to deserialize token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} TypedToken @2\nfailed to deobfuscate fabtoken token: failed unmarshalling token: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} TypedToken @2",
		},
		{
			name: "failed to deserialize fabtoken metadata",
			inputs: func() (*token2.TokensService, driver.TokenOutput, driver.TokenOutputMetadata, error) {
				ts, err := token2.NewTokensService(nil, pp, &mock.IdentityDeserializer{})
				if err != nil {
					return nil, nil, nil, err
				}
				tok := &actions.Output{}
				raw, err := tok.Serialize()
				if err != nil {
					return nil, nil, nil, err
				}
				return ts, raw, nil, nil
			},
			wantErr:       true,
			expectedError: "failed to deobfuscate: failed to deobfuscate comm token: failed to deobfuscate token: failed getting token output: failed to deserialize token: failed deserializing token: invalid token type [1]\nfailed to deobfuscate fabtoken token: failed unmarshalling token metadata: failed deserializing metadata: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name: "failed to deserialize fabtoken owner identity",
			inputs: func() (*token2.TokensService, driver.TokenOutput, driver.TokenOutputMetadata, error) {
				des := &mock.IdentityDeserializer{}
				des.RecipientsReturns(nil, errors.New("pineapple"))
				ts, err := token2.NewTokensService(nil, pp, des)
				if err != nil {
					return nil, nil, nil, err
				}
				tok := &actions.Output{}
				raw, err := tok.Serialize()
				if err != nil {
					return nil, nil, nil, err
				}

				meta := &actions.OutputMetadata{}
				metaRaw, err := meta.Serialize()
				if err != nil {
					return nil, nil, nil, err
				}

				return ts, raw, metaRaw, nil
			},
			wantErr:       true,
			expectedError: "failed to deobfuscate: failed to deobfuscate comm token: failed to deobfuscate token: failed getting token output: failed to deserialize token: failed deserializing token: invalid token type [1]\nfailed to deobfuscate fabtoken token: failed to get recipients: pineapple",
		},
		{
			name: "fabtoken output, cannot deserialize output",
			inputs: func() (*token2.TokensService, driver.TokenOutput, driver.TokenOutputMetadata, error) {
				des := &mock.IdentityDeserializer{}
				des.RecipientsReturns(nil, errors.New("pineapple"))
				ts, err := token2.NewTokensService(nil, pp, des)
				if err != nil {
					return nil, nil, nil, err
				}
				tok := &actions.Output{}
				raw, err := tok.Serialize()
				if err != nil {
					return nil, nil, nil, err
				}

				meta := &actions.OutputMetadata{}
				metaRaw, err := meta.Serialize()
				if err != nil {
					return nil, nil, nil, err
				}

				return ts, raw, metaRaw, nil
			},
			wantErr:       true,
			expectedError: "failed to deobfuscate: failed to deobfuscate comm token: failed to deobfuscate token: failed getting token output: failed to deserialize token: failed deserializing token: invalid token type [1]\nfailed to deobfuscate fabtoken token: failed to get recipients: pineapple",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, output, metadata, err := tt.inputs()
			assert.NoError(t, err)
			_, _, _, _, err = ts.Deobfuscate(output, metadata)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ts)
			}
		})
	}
}
