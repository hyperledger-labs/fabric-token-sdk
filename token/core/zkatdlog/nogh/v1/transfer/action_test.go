/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer_test

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	fabtokenv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAction_Validate(t *testing.T) {
	tests := []struct {
		name          string
		action        *transfer.Action
		wantErr       bool
		expectedError string
	}{
		{
			name:          "",
			action:        &transfer.Action{},
			wantErr:       true,
			expectedError: "invalid number of token inputs, expected at least 1",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{},
			},
			wantErr:       true,
			expectedError: "invalid number of token inputs, expected at least 1",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					nil,
				},
			},
			wantErr:       true,
			expectedError: "invalid input at index [0], empty input",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID:             nil,
						Token:          nil,
						UpgradeWitness: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's ID at index [0], it is empty",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID:             &token.ID{},
						Token:          nil,
						UpgradeWitness: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's ID at index [0], tx id is empty",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID:             &token.ID{TxId: "txid"},
						Token:          nil,
						UpgradeWitness: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's token at index [0], empty token",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: nil,
							Data:  nil,
						},
						UpgradeWitness: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input token at index [0]: token owner cannot be empty",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  nil,
						},
						UpgradeWitness: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input token at index [0]: token data cannot be empty",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: nil,
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid number of token outputs, expected at least 1",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's upgrade witness at index [0]: missing FabToken",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{},
						},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's upgrade witness at index [0]: missing FabToken.Owner",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner: []byte("owner"),
							},
						},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's upgrade witness at index [0]: missing FabToken.Type",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner: []byte("owner"),
								Type:  "type",
							},
						},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's upgrade witness at index [0]: missing FabToken.Quantity",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner:    []byte("owner"),
								Type:     "type",
								Quantity: "10",
							},
						},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid input's upgrade witness at index [0]: missing BlindingFactor",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner:    []byte("owner"),
								Type:     "type",
								Quantity: "10",
							},
							BlindingFactor: &math.Zr{},
						},
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid number of token outputs, expected at least 1",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner:    []byte("owner"),
								Type:     "type",
								Quantity: "10",
							},
							BlindingFactor: &math.Zr{},
						},
					},
				},
				Outputs: []*token2.Token{
					nil,
				},
			},
			wantErr:       true,
			expectedError: "invalid output token at index [0]",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner:    []byte("owner"),
								Type:     "type",
								Quantity: "10",
							},
							BlindingFactor: &math.Zr{},
						},
					},
				},
				Outputs: []*token2.Token{
					{},
				},
			},
			wantErr:       true,
			expectedError: "invalid output at index [0]: token data cannot be empty",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner:    []byte("owner"),
								Type:     "type",
								Quantity: "10",
							},
							BlindingFactor: &math.Zr{},
						},
					},
				},
				Outputs: []*token2.Token{
					{
						Owner: []byte("owner"),
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid output at index [0]: token data cannot be empty",
		},
		{
			name: "A Redeem action must have an issuer",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner:    []byte("owner"),
								Type:     "type",
								Quantity: "10",
							},
							BlindingFactor: &math.Zr{},
						},
					},
				},
				Outputs: []*token2.Token{
					{
						Owner: []byte(nil),
						Data:  &math.G1{},
					},
				},
				Issuer: []byte(nil),
			},
			wantErr:       true,
			expectedError: "Expected Issuer for a Redeem action",
		},
		{
			name: "",
			action: &transfer.Action{
				Inputs: []*transfer.ActionInput{
					{
						ID: &token.ID{TxId: "txid"},
						Token: &token2.Token{
							Owner: []byte("owner"),
							Data:  &math.G1{},
						},
						UpgradeWitness: &token2.UpgradeWitness{
							FabToken: &fabtokenv1.Output{
								Owner:    []byte("owner"),
								Type:     "type",
								Quantity: "10",
							},
							BlindingFactor: &math.Zr{},
						},
					},
				},
				Outputs: []*token2.Token{
					{
						Owner: []byte("owner"),
						Data:  &math.G1{},
					},
				},
				Issuer: []byte("issuer"),
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

func TestSerialization(t *testing.T) {
	action := randomAction(math.Curves[TestCurve], rand.Reader, t)
	raw, err := action.Serialize()
	require.NoError(t, err, "failed to serialize a new transfer action")

	action2 := &transfer.Action{}
	err = action2.Deserialize(raw)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action, action2, "deserialized action is not equal to the original one")

	raw2, err := action2.Serialize()
	require.NoError(t, err, "failed to serialize a new transfer action")

	action3 := &transfer.Action{}
	err = action3.Deserialize(raw2)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action2, action3, "deserialized action is not equal to the original one")
}

func TestAction_GetIssuer(t *testing.T) {
	issuerId := []byte("issuer")
	action := &transfer.Action{
		Issuer: issuerId,
	}
	issuer := action.GetIssuer()
	assert.True(t, issuer.Equal(issuerId), "unexpected issuer id in Action")
}

func BenchmarkActionMarshalling(b *testing.B) {
	curve := math.Curves[TestCurve]

	b.Run("With Protos", func(b *testing.B) {
		rand, err := curve.Rand()
		require.NoError(b, err, "failed to get random number")
		for range b.N {
			b.StopTimer()
			action := randomAction(curve, rand, b)
			b.StartTimer()
			_, err = action.Serialize()
			require.NoError(b, err, "failed to serialize a new transfer action")
		}
	})

	b.Run("With json", func(b *testing.B) {
		rand, err := curve.Rand()
		require.NoError(b, err, "failed to get random number")
		for range b.N {
			b.StopTimer()
			action := randomAction(curve, rand, b)
			b.StartTimer()
			_, err = json.Marshal(action)
			require.NoError(b, err, "failed to serialize a new transfer action")
		}
	})
}

func getRandomBytes(b require.TestingT, len int) []byte {
	key := make([]byte, len)
	_, err := rand.Read(key)
	require.NoError(b, err, "error getting random bytes")

	return key
}

func randomAction(curve *math.Curve, rand io.Reader, b require.TestingT) *transfer.Action {
	// generate an action at random
	tokenIDs := []*token.ID{
		{
			TxId:  base64.StdEncoding.EncodeToString(getRandomBytes(b, 32)),
			Index: 0,
		},
		{
			TxId:  base64.StdEncoding.EncodeToString(getRandomBytes(b, 32)),
			Index: 0,
		},
	}
	inputToken := []*token2.Token{
		{
			Owner: getRandomBytes(b, 32),
			Data:  curve.GenG1.Mul(curve.NewRandomZr(rand)),
		},
		{
			Owner: getRandomBytes(b, 32),
			Data:  curve.GenG1.Mul(curve.NewRandomZr(rand)),
		},
	}
	commitments := []*math.G1{
		curve.GenG1.Mul(curve.NewRandomZr(rand)),
		curve.GenG1.Mul(curve.NewRandomZr(rand)),
	}
	owners := [][]byte{
		getRandomBytes(b, 32),
		getRandomBytes(b, 32),
	}
	proof := getRandomBytes(b, 32)
	action, err := transfer.NewTransfer(tokenIDs, inputToken, commitments, owners, proof)
	require.NoError(b, err, "failed to create a new transfer action")
	action.Metadata = map[string][]byte{
		"key1": getRandomBytes(b, 32),
		"key2": getRandomBytes(b, 32),
	}
	action.Issuer = getRandomBytes(b, 32)

	return action
}
