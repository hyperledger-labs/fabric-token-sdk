/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	fabtokenv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestAction_Validate(t *testing.T) {
	tests := []struct {
		name          string
		action        *Action
		wantErr       bool
		expectedError string
	}{
		{
			name:          "",
			action:        &Action{},
			wantErr:       true,
			expectedError: "invalid number of token inputs, expected at least 1",
		},
		{
			name: "",
			action: &Action{
				Inputs: []*ActionInput{},
			},
			wantErr:       true,
			expectedError: "invalid number of token inputs, expected at least 1",
		},
		{
			name: "",
			action: &Action{
				Inputs: []*ActionInput{
					nil,
				},
			},
			wantErr:       true,
			expectedError: "invalid input at index [0], empty input",
		},
		{
			name: "",
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			action: &Action{
				Inputs: []*ActionInput{
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
			name: "",
			action: &Action{
				Inputs: []*ActionInput{
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
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSerialization(t *testing.T) {
	action := randomAction(math.Curves[math.BN254], rand.Reader, t)
	raw, err := action.Serialize()
	assert.NoError(t, err, "failed to serialize a new transfer action")

	action2 := &Action{}
	err = action2.Deserialize(raw)
	assert.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action, action2, "deserialized action is not equal to the original one")

	raw2, err := action2.Serialize()
	assert.NoError(t, err, "failed to serialize a new transfer action")

	action3 := &Action{}
	err = action3.Deserialize(raw2)
	assert.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action2, action3, "deserialized action is not equal to the original one")
}

func BenchmarkActionMarshalling(b *testing.B) {
	curve := math.Curves[math.BN254]

	b.Run("With Protos", func(b *testing.B) {
		rand, err := curve.Rand()
		assert.NoError(b, err, "failed to get random number")
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			action := randomAction(curve, rand, b)
			b.StartTimer()
			_, err = action.Serialize()
			assert.NoError(b, err, "failed to serialize a new transfer action")
		}
	})

	b.Run("With json", func(b *testing.B) {
		rand, err := curve.Rand()
		assert.NoError(b, err, "failed to get random number")
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			action := randomAction(curve, rand, b)
			b.StartTimer()
			_, err = json.Marshal(action)
			assert.NoError(b, err, "failed to serialize a new transfer action")
		}
	})
}

func getRandomBytes(b assert.TestingT, len int) []byte {
	key := make([]byte, len)
	_, err := rand.Read(key)
	assert.NoError(b, err, "error getting random bytes")
	return key
}

func randomAction(curve *math.Curve, rand io.Reader, b assert.TestingT) *Action {
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
	action, err := NewTransfer(tokenIDs, inputToken, commitments, owners, proof)
	assert.NoError(b, err, "failed to create a new transfer action")
	action.Metadata = map[string][]byte{
		"key1": getRandomBytes(b, 32),
		"key2": getRandomBytes(b, 32),
	}
	return action
}
