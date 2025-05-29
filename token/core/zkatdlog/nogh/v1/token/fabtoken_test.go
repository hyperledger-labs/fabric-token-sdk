/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/stretchr/testify/assert"
)

func TestParseFabtokenToken(t *testing.T) {
	nilGetTokFunc := func() (*actions.Output, []byte, error) {
		return nil, nil, nil
	}
	tests := []struct {
		name             string
		tok              func() (*actions.Output, []byte, error)
		precision        uint64
		maxPrecision     uint64
		wantErr          bool
		expectedError    string
		expectedQuantity uint64
	}{
		{
			name:          "precision is langer than maxPrecision",
			tok:           nilGetTokFunc,
			precision:     10,
			maxPrecision:  5,
			wantErr:       true,
			expectedError: "unsupported precision [10], max [5]",
		},
		{
			name:          "invalid tok",
			tok:           nilGetTokFunc,
			precision:     5,
			maxPrecision:  10,
			wantErr:       true,
			expectedError: "failed to unmarshal fabtoken: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name: "invalid tok 2",
			tok: func() (*actions.Output, []byte, error) {
				return nil, []byte{}, nil
			},
			precision:     5,
			maxPrecision:  10,
			wantErr:       true,
			expectedError: "failed to unmarshal fabtoken: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: syntax error: sequence truncated",
		},
		{
			name: "invalid tok 3",
			tok: func() (*actions.Output, []byte, error) {
				return nil, []byte{0, 1, 2}, nil
			},
			precision:     5,
			maxPrecision:  10,
			wantErr:       true,
			expectedError: "failed to unmarshal fabtoken: failed deserializing token: failed unmarshalling token: failed to unmarshal to TypedToken: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} TypedToken @2",
		},
		{
			name: "invalid quantity",
			tok: func() (*actions.Output, []byte, error) {
				output := &actions.Output{
					Owner:    nil,
					Type:     "",
					Quantity: "",
				}
				raw, err := output.Serialize()
				return output, raw, err
			},
			precision:     5,
			maxPrecision:  10,
			wantErr:       true,
			expectedError: "failed to create quantity: invalid input [,5]",
		},
		{
			name: "success",
			tok: func() (*actions.Output, []byte, error) {
				output := &actions.Output{
					Owner:    nil,
					Type:     "",
					Quantity: "10",
				}
				raw, err := output.Serialize()
				return output, raw, err
			},
			precision:        5,
			maxPrecision:     10,
			wantErr:          false,
			expectedQuantity: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, tokBytes, err := tt.tok()
			assert.NoError(t, err)
			output, quantity, err := ParseFabtokenToken(tokBytes, tt.precision, tt.maxPrecision)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tok, output)
				assert.Equal(t, tt.expectedQuantity, quantity)
			}
		})
	}
}
