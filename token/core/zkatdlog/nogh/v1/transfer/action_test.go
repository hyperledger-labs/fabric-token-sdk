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
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	fabtokenv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/actions"
	zkatmath "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/math"
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
			expectedError: "invalid input at index [0], empty input: invalid input, empty input",
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
			expectedError: "invalid input's ID at index [0], it is empty: invalid input's ID, it is empty",
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
			expectedError: "invalid input's ID at index [0], tx id is empty: invalid input's ID, tx id is empty",
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
			expectedError: "invalid input's token at index [0], empty token: invalid input's token, empty token",
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
			expectedError: "invalid output token at index [0]: invalid output token, empty token",
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
			expectedError: "expected issuer for a redeem action",
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

	action2, err := transfer.NewActionFromProtos(raw)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action, action2, "deserialized action is not equal to the original one")

	raw2, err := action2.Serialize()
	require.NoError(t, err, "failed to serialize a new transfer action")

	action3, err := transfer.NewActionFromProtos(raw2)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action2, action3, "deserialized action is not equal to the original one")
}

func TestNewTransfer(t *testing.T) {
	action := randomAction(math.Curves[TestCurve], rand.Reader, t)
	assert.NotNil(t, action)

	assert.Equal(t, 2, action.NumInputs())
	assert.NotNil(t, action.GetInputs())
	assert.Nil(t, action.GetSerialNumbers())
	assert.Equal(t, 2, action.NumOutputs())
	assert.False(t, action.IsRedeemAt(0))
	assert.False(t, action.IsGraphHiding())
	assert.NotNil(t, action.GetMetadata())
	assert.NotNil(t, action.InputTokens())

	serializedInputs, err := action.GetSerializedInputs()
	assert.NoError(t, err)
	assert.Len(t, serializedInputs, 2)

	outputs := action.GetOutputs()
	assert.Len(t, outputs, 2)

	serializedOutputs, err := action.GetSerializedOutputs()
	assert.NoError(t, err)
	assert.Len(t, serializedOutputs, 2)
}

func TestNewActionFromProtos(t *testing.T) {
	// create a random action
	action := randomAction(math.Curves[TestCurve], rand.Reader, t)
	assert.NotNil(t, action)

	// serialize it
	raw, err := action.Serialize()
	assert.NoError(t, err)

	// deserialize it
	action2 := &transfer.Action{}
	err = action2.Deserialize(raw)
	assert.NoError(t, err)

	// check that the deserialized action is equal to the original one
	assert.Equal(t, action, action2)

	// create a new action from protos
	action3, err := transfer.NewActionFromProtos(raw)
	assert.NoError(t, err)
	assert.NotNil(t, action3)
	assert.Equal(t, action2, action3)
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

func TestExtraSigners(t *testing.T) {
	action := &transfer.Action{
		Outputs: []*token2.Token{
			{
				Owner: []byte(nil), // redeem
			},
		},
		Issuer: []byte("issuer"),
	}
	assert.Nil(t, action.ExtraSigners())
	assert.True(t, action.IsRedeem())
	assert.True(t, action.IsRedeemAt(0))
	assert.Equal(t, []byte("issuer"), action.GetIssuer().Bytes())
	assert.Nil(t, action.GetSerialNumbers())
}

func TestAction_ToProtos(t *testing.T) {
	c := math.Curves[TestCurve]
	ai := &transfer.ActionInput{
		ID: &token.ID{TxId: "txid", Index: 1},
		Token: &token2.Token{
			Owner: []byte("owner"),
			Data:  c.GenG1.Copy(),
		},
	}
	p, err := ai.ToProtos()
	assert.NoError(t, err)
	assert.NotNil(t, p)

	ai2 := &transfer.ActionInput{}
	err = ai2.FromProtos(p)
	assert.NoError(t, err)
	assert.Equal(t, ai.ID, ai2.ID)
	assert.Equal(t, ai.Token.Owner, ai2.Token.Owner)
	assert.True(t, ai.Token.Data.Equals(ai2.Token.Data))

	// test with upgrade witness
	ai.UpgradeWitness = &token2.UpgradeWitness{
		FabToken: &fabtokenv1.Output{
			Owner:    []byte("owner"),
			Type:     "type",
			Quantity: "10",
		},
		BlindingFactor: c.NewZrFromInt(1),
	}
	p, err = ai.ToProtos()
	assert.NoError(t, err)
	assert.NotNil(t, p)

	err = ai2.FromProtos(p)
	assert.NoError(t, err)
	assert.Equal(t, ai.UpgradeWitness.FabToken.Owner, ai2.UpgradeWitness.FabToken.Owner)
	assert.Equal(t, ai.UpgradeWitness.FabToken.Type, ai2.UpgradeWitness.FabToken.Type)
	assert.Equal(t, ai.UpgradeWitness.FabToken.Quantity, ai2.UpgradeWitness.FabToken.Quantity)
	assert.True(t, ai.UpgradeWitness.BlindingFactor.Equals(ai2.UpgradeWitness.BlindingFactor))
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
	action, err := transfer.NewAction(tokenIDs, inputToken, commitments, owners, proof)
	require.NoError(b, err, "failed to create a new transfer action")
	action.Metadata = map[string][]byte{
		"key1": getRandomBytes(b, 32),
		"key2": getRandomBytes(b, 32),
	}
	action.Issuer = getRandomBytes(b, 32)

	return action
}

func TestAction_Deserialize_ErrorPaths(t *testing.T) {
	// 1. Invalid proto bytes
	action := &transfer.Action{}
	err := action.Deserialize([]byte("invalid"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize transfer action")

	// 2. Invalid version
	protoAction := &actions.TransferAction{
		Version: 100, // Invalid version
	}
	raw, err := proto.Marshal(protoAction)
	assert.NoError(t, err)
	err = action.Deserialize(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected [1], got [100]: invalid transfer version")

	// 3. Invalid inputs (FromProtos failure)
	protoAction = &actions.TransferAction{
		Version: 1, // ProtocolV1
		Inputs: []*actions.TransferActionInput{
			{
				Input: &actions.Token{
					Data: &zkatmath.G1{Raw: []byte("invalid")},
				},
			},
		},
	}
	raw, err = proto.Marshal(protoAction)
	assert.NoError(t, err)
	err = action.Deserialize(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed unmarshalling inputs")

	// 4. Invalid outputs (FromG1Proto failure)
	protoAction = &actions.TransferAction{
		Version: 1, // ProtocolV1
		Outputs: []*actions.TransferActionOutput{
			{
				Token: &actions.Token{
					Data: &zkatmath.G1{Raw: []byte("invalid")},
				},
			},
		},
	}
	raw, err = proto.Marshal(protoAction)
	assert.NoError(t, err)
	err = action.Deserialize(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize output")
}

func TestAction_Deserialize_ExtraBranches(t *testing.T) {
	// Test with nil output and nil output.Token
	protoAction := &actions.TransferAction{
		Version: 1, // ProtocolV1
		Outputs: []*actions.TransferActionOutput{
			nil,
			{
				Token: nil,
			},
		},
	}
	raw, err := proto.Marshal(protoAction)
	assert.NoError(t, err)
	action := &transfer.Action{}
	err = action.Deserialize(raw)
	assert.NoError(t, err)
	assert.Len(t, action.Outputs, 2)
	assert.Nil(t, action.Outputs[0])
	assert.Nil(t, action.Outputs[1])
}

func TestAction_SerializeOutputAt(t *testing.T) {
	c := math.Curves[TestCurve]
	action := &transfer.Action{
		Outputs: []*token2.Token{
			{
				Owner: []byte("owner"),
				Data:  c.GenG1.Copy(),
			},
		},
	}
	raw, err := action.SerializeOutputAt(0)
	assert.NoError(t, err)
	assert.NotNil(t, raw)
}

func TestAction_GetSerializedInputs(t *testing.T) {
	c := math.Curves[TestCurve]
	action := &transfer.Action{
		Inputs: []*transfer.ActionInput{
			{
				ID: &token.ID{TxId: "txid", Index: 0},
				Token: &token2.Token{
					Owner: []byte("owner"),
					Data:  c.GenG1.Copy(),
				},
			},
		},
	}
	res, err := action.GetSerializedInputs()
	assert.NoError(t, err)
	assert.Len(t, res, 1)
}
