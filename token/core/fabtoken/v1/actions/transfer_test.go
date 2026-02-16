/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package actions

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func TestTransferAction_Validate(t *testing.T) {
	tests := []struct {
		name          string
		action        *TransferAction
		wantErr       bool
		expectedError string
	}{
		{
			name:          "",
			action:        &TransferAction{},
			wantErr:       true,
			expectedError: "invalid number of token inputs, expected at least 1",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{},
			},
			wantErr:       true,
			expectedError: "invalid number of token inputs, expected at least 1",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					nil,
				},
			},
			wantErr:       true,
			expectedError: "invalid input at index [0], empty input",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID:    nil,
						Input: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's ID at index [0], it is empty",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID:    &token.ID{},
						Input: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's ID at index [0], tx id is empty",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID:    &token.ID{TxId: "txid"},
						Input: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's token at index [0], empty token",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID:    &token.ID{TxId: "txid"},
						Input: &Output{},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input token at index [0]: token owner cannot be empty",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Input: &Output{
							Owner:    []byte("owner"),
							Type:     "",
							Quantity: "",
						},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input token at index [0]: token quantity cannot be empty",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Input: &Output{
							Owner:    []byte("owner"),
							Type:     "type",
							Quantity: "",
						},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input token at index [0]: token quantity cannot be empty",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Input: &Output{
							Owner:    []byte("owner"),
							Type:     "type",
							Quantity: "11",
						},
					},
				},
				Outputs: []*Output{
					nil,
				},
			},
			wantErr:       true,
			expectedError: "invalid output at index [0], empty output",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Input: &Output{
							Owner:    []byte("owner"),
							Type:     "type",
							Quantity: "11",
						},
					},
				},
				Outputs: []*Output{
					{},
				},
			},
			wantErr:       true,
			expectedError: "invalid output's type at index [0], output type is empty",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Input: &Output{
							Owner:    []byte("owner"),
							Type:     "type",
							Quantity: "11",
						},
					},
				},
				Outputs: []*Output{
					{
						Type: "type",
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid output's quantity at index [0], output quantity is empty",
		},
		{
			name: "",
			action: &TransferAction{
				Inputs: []*TransferActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Input: &Output{
							Owner:    []byte("owner"),
							Type:     "type",
							Quantity: "11",
						},
					},
				},
				Outputs: []*Output{
					{
						Owner:    []byte("owner"),
						Type:     "type",
						Quantity: "11",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTransferAction_Serialization(t *testing.T) {
	action := &TransferAction{
		Inputs: []*TransferActionInput{
			{
				ID: &token.ID{
					TxId:  "txid",
					Index: 0,
				},
				Input: &Output{
					Owner:    []byte("owner"),
					Type:     "type",
					Quantity: "11",
				},
			},
		},
		Outputs: []*Output{
			{
				Type:     "type",
				Quantity: "11",
				Owner:    []byte("owner"),
			},
		},
		Metadata: map[string][]byte{
			"metadata": []byte("{\"foo\":\"bar\"}"),
		},
		Issuer: nil,
	}
	raw, err := action.Serialize()
	require.NoError(t, err, "failed to serialize a new transfer action")

	action2 := &TransferAction{}
	err = action2.Deserialize(raw)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action, action2, "deserialized action is not equal to the original one")

	raw2, err := action2.Serialize()
	require.NoError(t, err, "failed to serialize a new transfer action")

	action3 := &TransferAction{}
	err = action3.Deserialize(raw2)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action2, action3, "deserialized action is not equal to the original one")
}

func TestTransferAction_GetIssuer(t *testing.T) {
	issuerId := []byte("issuer")
	action := &TransferAction{
		Issuer: issuerId,
	}
	issuer := action.GetIssuer()
	assert.True(t, issuer.Equal(issuerId), "unexpected issuer id in TransferAction")
}
