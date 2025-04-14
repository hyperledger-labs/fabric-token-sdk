/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"testing"

	math "github.com/IBM/mathlib"
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
	pp, err = v1.Setup(64, nil, math.FP256BN_AMCL)
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
			token.Data = math.Curves[math.FP256BN_AMCL].NewG1()
		}
		raw, err := token.Serialize()
		assert.NoError(f, err)
		assert.NotNil(t, raw)

		token2 := &token2.Token{}
		err = token2.Deserialize(raw)
		if err != nil {
			t.Errorf("failed to deserialize token [owner: %s, putData: %v]: [%v]", owner, putData, err)
		}
		assert.Equal(t, len(token.Owner), len(token2.Owner), "owner mismatch [owner: %s, putData: %v]", owner, putData)
		assert.Equal(t, token.Data, token2.Data)
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
		name             string
		values           []uint64
		tokenType        token3.Type
		pp               []*math.G1
		curve            *math.Curve
		wantErr          bool
		expectedError    string
		expectedQuantity uint64
	}{
		{
			name:          "curve is nil",
			wantErr:       true,
			expectedError: "cannot get tokens with witness: please initialize curve",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := token2.GetTokensWithWitness(
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
