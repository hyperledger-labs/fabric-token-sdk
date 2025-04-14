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
