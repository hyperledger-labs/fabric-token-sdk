/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	fabtokenv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/comm"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestToClear(t *testing.T) {
	var (
		inf   *token2.Metadata
		token *token2.Token
		pp    *v1.PublicParams
		err   error
	)
	pp, err = v1.Setup(64, nil, math.BN254)
	assert.NoError(t, err)
	c := math.Curves[pp.Curve]
	rand, err := c.Rand()
	assert.NoError(t, err)
	inf = &token2.Metadata{
		Value:          c.NewZrFromInt(50),
		Type:           "ABC",
		BlindingFactor: c.NewRandomZr(rand),
	}
	token = &token2.Token{}
	token.Data = c.NewG1()
	token.Data.Add(pp.PedersenGenerators[1].Mul(inf.Value))
	token.Data.Add(pp.PedersenGenerators[2].Mul(inf.BlindingFactor))
	token.Data.Add(pp.PedersenGenerators[0].Mul(c.HashToZr([]byte("ABC"))))
	tok, err := token.ToClear(inf, pp)
	assert.NoError(t, err)
	assert.Equal(t, token3.Type("ABC"), tok.Type)
	assert.Equal(t, "0x"+inf.Value.String(), tok.Quantity)
}

func FuzzSerialization(f *testing.F) {
	testcases := [][]any{
		{[]byte("Alice"), false},
		{[]byte("Charlie"), true},
	}
	for _, tc := range testcases {
		f.Add(tc[0].([]byte), tc[1].(bool))
	}
	f.Fuzz(func(t *testing.T, owner []byte, putData bool) {
		token := &token2.Token{
			Owner: owner,
		}
		if putData {
			token.Data = math.Curves[math.BN254].NewG1()
		}
		raw, err := token.Serialize()
		assert.NoError(f, err)
		assert.NotNil(t, raw)

		desToken := &token2.Token{}
		err = desToken.Deserialize(raw)
		if err != nil {
			t.Errorf("failed to deserialize metadata [owner: %s, putData: %v]: [%v]", owner, putData, err)
		}
		assert.Equal(t, len(token.Owner), len(desToken.Owner), "owner mismatch [owner: %s, putData: %v]", owner, putData)
		assert.Equal(t, token.Data, desToken.Data)
	})
}

func TestTokenGetOwner(t *testing.T) {
	token := &token2.Token{
		Owner: []byte("Alice"),
	}
	assert.Equal(t, token.GetOwner(), token.Owner)
}

func TestTokenIsRedeem(t *testing.T) {
	token := &token2.Token{
		Owner: []byte("Alice"),
	}
	assert.False(t, token.IsRedeem())

	token = &token2.Token{}
	assert.True(t, token.IsRedeem())

	token = &token2.Token{
		Owner: []byte{},
	}
	assert.True(t, token.IsRedeem())
}

func TestGetTokensWithWitness(t *testing.T) {
	tests := []struct {
		name          string
		values        []uint64
		tokenType     token3.Type
		pp            []*math.G1
		curve         *math.Curve
		validate      func([]*math.G1, []*token2.Metadata) error
		wantErr       bool
		expectedError string
	}{
		{
			name:          "curve is nil",
			wantErr:       true,
			expectedError: "cannot get tokens with witness: please initialize curve",
		},
		{
			name:    "curve is not nil",
			curve:   math.Curves[math.FP256BN_AMCL],
			wantErr: false,
			validate: func(tokens []*math.G1, data []*token2.Metadata) error {
				if len(tokens) != 0 {
					return errors.New("tokens should be empty")
				}
				if len(data) != 0 {
					return errors.New("tokens should be empty")
				}
				return nil
			},
		},
		{
			name:          "number of generators is not equal to number of vector elements",
			values:        []uint64{10},
			tokenType:     "token type",
			pp:            nil,
			curve:         math.Curves[math.FP256BN_AMCL],
			wantErr:       true,
			expectedError: "cannot get tokens with witness: failed to compute token [0]: number of generators is not equal to number of vector elements, [0]!=[3]",
		},
		{
			name:      "success",
			values:    []uint64{10},
			tokenType: "token type",
			pp: []*math.G1{
				math.Curves[math.FP256BN_AMCL].NewG1(),
				math.Curves[math.FP256BN_AMCL].NewG1(),
				math.Curves[math.FP256BN_AMCL].NewG1(),
			},
			curve:   math.Curves[math.FP256BN_AMCL],
			wantErr: false,
			validate: func(toks []*math.G1, data []*token2.Metadata) error {
				if len(toks) != 1 {
					return errors.New("one token was expected")
				}
				if len(data) != 1 {
					return errors.New("one data was expected")
				}
				c := math.Curves[math.FP256BN_AMCL]
				pp := []*math.G1{
					math.Curves[math.FP256BN_AMCL].NewG1(),
					math.Curves[math.FP256BN_AMCL].NewG1(),
					math.Curves[math.FP256BN_AMCL].NewG1(),
				}

				for i, token := range toks {
					hash := c.HashToZr([]byte(data[i].Type))
					tok, err := token2.Commit(
						[]*math.Zr{
							hash,
							data[i].Value,
							data[i].BlindingFactor,
						},
						pp,
						c,
					)
					if err != nil {
						return err
					}
					if !token.Equals(tok) {
						return errors.New("token does not match")
					}
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g1s, witnesses, err := token2.GetTokensWithWitness(
				tt.values,
				tt.tokenType,
				tt.pp,
				tt.curve,
			)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, tt.validate(g1s, witnesses))
			}
		})
	}
}

func TestTokenValidate(t *testing.T) {
	tests := []struct {
		name          string
		token         func() (*token2.Token, error)
		owner         bool
		wantErr       bool
		expectedError string
	}{
		{
			name:  "owner is nil",
			owner: true,
			token: func() (*token2.Token, error) {
				return &token2.Token{}, nil
			},
			wantErr:       true,
			expectedError: "token owner cannot be empty",
		},
		{
			name:  "owner is empty",
			owner: true,
			token: func() (*token2.Token, error) {
				return &token2.Token{Owner: []byte{}}, nil
			},
			wantErr:       true,
			expectedError: "token owner cannot be empty",
		},
		{
			name:  "data is empty",
			owner: true,
			token: func() (*token2.Token, error) {
				return &token2.Token{Owner: []byte("owner")}, nil
			},
			wantErr:       true,
			expectedError: "token data cannot be empty",
		},
		{
			name:  "data is empty with no owner",
			owner: false,
			token: func() (*token2.Token, error) {
				return &token2.Token{}, nil
			},
			wantErr:       true,
			expectedError: "token data cannot be empty",
		},
		{
			name:  "valid with no owner",
			owner: false,
			token: func() (*token2.Token, error) {
				return &token2.Token{Data: &math.G1{}}, nil
			},
			wantErr: false,
		},
		{
			name:  "valid with owner",
			owner: true,
			token: func() (*token2.Token, error) {
				return &token2.Token{Owner: []byte("owner"), Data: &math.G1{}}, nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := tt.token()
			assert.NoError(t, err)
			err = tok.Validate(tt.owner)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenDeserialize(t *testing.T) {
	tests := []struct {
		name          string
		token         func() (*token2.Token, []byte, error)
		owner         bool
		wantErr       bool
		expectedError string
	}{
		{
			name:  "nil raw",
			owner: true,
			token: func() (*token2.Token, []byte, error) {
				return nil, nil, nil
			},
			wantErr:       true,
			expectedError: "failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name:  "empty raw",
			owner: true,
			token: func() (*token2.Token, []byte, error) {
				return nil, []byte{}, nil
			},
			wantErr:       true,
			expectedError: "failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name:  "invalid raw",
			owner: true,
			token: func() (*token2.Token, []byte, error) {
				return nil, []byte{0, 1, 2, 3}, nil
			},
			wantErr:       true,
			expectedError: "failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} TypedToken @2",
		},
		{
			name:  "invalid token type",
			owner: true,
			token: func() (*token2.Token, []byte, error) {
				raw, err := tokens.WrapWithType(-1, []byte{0, 1, 2, 3})
				return nil, raw, err
			},
			wantErr:       true,
			expectedError: "failed deserializing token: invalid token type [-1]",
		},
		{
			name:  "valid token raw, nil",
			owner: true,
			token: func() (*token2.Token, []byte, error) {
				raw, err := tokens.WrapWithType(comm.Type, nil)
				return &token2.Token{}, raw, err
			},
			wantErr: false,
		},
		{
			name:  "invalid token raw, invalid",
			owner: true,
			token: func() (*token2.Token, []byte, error) {
				raw, err := tokens.WrapWithType(comm.Type, []byte{0, 1, 2, 3})
				return nil, raw, err
			},
			wantErr:       true,
			expectedError: "failed unmarshalling token: proto: cannot parse invalid wire-format data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, raw, err := tt.token()
			assert.NoError(t, err)
			tok2 := &token2.Token{}
			err = tok2.Deserialize(raw)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tok, tok2)
			}
		})
	}
}

func TestMetadataDeserialize(t *testing.T) {
	tests := []struct {
		name          string
		metadata      func() (*token2.Metadata, []byte, error)
		owner         bool
		wantErr       bool
		expectedError string
	}{
		{
			name:  "nil raw",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				return nil, nil, nil
			},
			wantErr:       true,
			expectedError: "failed to deserialize metadata: failed unmarshalling metadata: failed to unmarshal to TypedMetadata: asn1: syntax error: sequence truncated",
		},
		{
			name:  "empty raw",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				return nil, []byte{}, nil
			},
			wantErr:       true,
			expectedError: "failed to deserialize metadata: failed unmarshalling metadata: failed to unmarshal to TypedMetadata: asn1: syntax error: sequence truncated",
		},
		{
			name:  "invalid raw",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				return nil, []byte{0, 1, 2, 3}, nil
			},
			wantErr:       true,
			expectedError: "failed to deserialize metadata: failed unmarshalling metadata: failed to unmarshal to TypedMetadata: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} TypedMetadata @2",
		},
		{
			name:  "invalid metadata type",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				raw, err := tokens.WrapWithType(-1, []byte{0, 1, 2, 3})
				return nil, raw, err
			},
			wantErr:       true,
			expectedError: "failed to deserialize metadata: invalid metadata type [-1]",
		},
		{
			name:  "valid metadata raw, nil",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				raw, err := tokens.WrapWithType(comm.Type, nil)
				return &token2.Metadata{}, raw, err
			},
			wantErr: false,
		},
		{
			name:  "invalid metadata raw, invalid",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				raw, err := tokens.WrapWithType(comm.Type, []byte{0, 1, 2, 3})
				return nil, raw, err
			},
			wantErr:       true,
			expectedError: "failed to deserialize metadata: proto: cannot parse invalid wire-format data",
		},
		{
			name:  "invalid metadata raw, invalid",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				raw, err := tokens.WrapWithType(comm.Type, []byte{0, 1, 2, 3})
				return nil, raw, err
			},
			wantErr:       true,
			expectedError: "failed to deserialize metadata: proto: cannot parse invalid wire-format data",
		},
		{
			name:  "valid metadata",
			owner: true,
			metadata: func() (*token2.Metadata, []byte, error) {
				c := math.Curves[math.BN254]
				rand, err := c.Rand()
				assert.NoError(t, err)
				metadata := &token2.Metadata{
					Type:           "token type",
					Value:          c.NewRandomZr(rand),
					BlindingFactor: c.NewRandomZr(rand),
					Issuer:         []byte("issuer"),
				}
				raw, err := metadata.Serialize()
				return metadata, raw, err
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, raw, err := tt.metadata()
			assert.NoError(t, err)
			metadata2 := &token2.Metadata{}
			err = metadata2.Deserialize(raw)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, metadata, metadata2)
			}
		})
	}
}

func TestUpgradeWitnessValidate(t *testing.T) {
	tests := []struct {
		name          string
		token         func() (*token2.UpgradeWitness, error)
		wantErr       bool
		expectedError string
	}{
		{
			name: "missing FabToken",
			token: func() (*token2.UpgradeWitness, error) {
				return &token2.UpgradeWitness{}, nil
			},
			wantErr:       true,
			expectedError: "missing FabToken",
		},
		{
			name: "missing FabToken.Owner",
			token: func() (*token2.UpgradeWitness, error) {
				return &token2.UpgradeWitness{
					FabToken:       &fabtokenv1.Output{},
					BlindingFactor: nil,
				}, nil
			},
			wantErr:       true,
			expectedError: "missing FabToken.Owner",
		},
		{
			name: "missing FabToken.Type",
			token: func() (*token2.UpgradeWitness, error) {
				return &token2.UpgradeWitness{
					FabToken: &fabtokenv1.Output{
						Owner:    []byte("owner"),
						Type:     "",
						Quantity: "",
					},
					BlindingFactor: nil,
				}, nil
			},
			wantErr:       true,
			expectedError: "missing FabToken.Type",
		},
		{
			name: "missing FabToken.Quantity",
			token: func() (*token2.UpgradeWitness, error) {
				return &token2.UpgradeWitness{
					FabToken: &fabtokenv1.Output{
						Owner:    []byte("owner"),
						Type:     "token type",
						Quantity: "",
					},
					BlindingFactor: nil,
				}, nil
			},
			wantErr:       true,
			expectedError: "missing FabToken.Quantity",
		},
		{
			name: "missing BlindingFactor",
			token: func() (*token2.UpgradeWitness, error) {
				return &token2.UpgradeWitness{
					FabToken: &fabtokenv1.Output{
						Owner:    []byte("owner"),
						Type:     "token type",
						Quantity: "quantity",
					},
					BlindingFactor: nil,
				}, nil
			},
			wantErr:       true,
			expectedError: "missing BlindingFactor",
		},
		{
			name: "success",
			token: func() (*token2.UpgradeWitness, error) {
				return &token2.UpgradeWitness{
					FabToken: &fabtokenv1.Output{
						Owner:    []byte("owner"),
						Type:     "token type",
						Quantity: "quantity",
					},
					BlindingFactor: &math.Zr{},
				}, nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := tt.token()
			assert.NoError(t, err)
			err = tok.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
