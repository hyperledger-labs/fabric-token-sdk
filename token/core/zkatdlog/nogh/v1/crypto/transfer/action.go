/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Action specifies a transfer of one or more tokens
type Action struct {
	// Inputs specify the identifiers in of the tokens to be spent
	Inputs []*token2.ID
	// InputCommitments are the PedersenCommitments in the inputs
	InputTokens         []*token.Token
	InputUpgradeWitness []*token.UpgradeWitness
	// OutputTokens are the new tokens resulting from the transfer
	OutputTokens []*token.Token
	// ZK Proof that shows that the transfer is correct
	Proof []byte
	// Metadata contains the transfer action's metadata
	Metadata map[string][]byte
}

// NewTransfer returns the Action that matches the passed arguments
func NewTransfer(inputs []*token2.ID, inputToken []*token.Token, outputs []*math.G1, owners [][]byte, proof []byte) (*Action, error) {
	if len(outputs) != len(owners) {
		return nil, errors.Errorf("number of recipients [%d] does not match number of outputs [%d]", len(outputs), len(owners))
	}
	if len(inputs) != len(inputToken) {
		return nil, errors.Errorf("number of inputs [%d] does not match number of input tokens [%d]", len(inputs), len(inputToken))
	}
	tokens := make([]*token.Token, len(owners))
	for i, o := range outputs {
		tokens[i] = &token.Token{Data: o, Owner: owners[i]}
	}
	return &Action{
		Inputs:       inputs,
		InputTokens:  inputToken,
		OutputTokens: tokens,
		Proof:        proof,
		Metadata:     map[string][]byte{},
	}, nil
}

func (t *Action) NumInputs() int {
	return len(t.Inputs)
}

// GetInputs returns the inputs in the Action
func (t *Action) GetInputs() []*token2.ID {
	return t.Inputs
}

func (t *Action) GetSerializedInputs() ([][]byte, error) {
	var res [][]byte
	for i, token := range t.InputTokens {
		if w := t.InputUpgradeWitness[i]; w != nil {
			ser, err := w.FabToken.Serialize()
			if err != nil {
				return nil, err
			}
			res = append(res, ser)
			continue
		}
		r, err := token.Serialize()
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (t *Action) GetSerialNumbers() []string {
	return nil
}

// NumOutputs returns the number of outputs in the Action
func (t *Action) NumOutputs() int {
	return len(t.OutputTokens)
}

// GetOutputs returns the outputs in the Action
func (t *Action) GetOutputs() []driver.Output {
	res := make([]driver.Output, len(t.OutputTokens))
	for i, outputToken := range t.OutputTokens {
		res[i] = outputToken
	}
	return res
}

// IsRedeemAt checks if output in the Action at the passed index is redeemed
func (t *Action) IsRedeemAt(index int) bool {
	return t.OutputTokens[index].IsRedeem()
}

// SerializeOutputAt marshals the output in the Action at the passed index
func (t *Action) SerializeOutputAt(index int) ([]byte, error) {
	return t.OutputTokens[index].Serialize()
}

// Serialize marshals the Action
func (t *Action) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// GetProof returns the proof in the Action
func (t *Action) GetProof() []byte {
	return t.Proof
}

// Deserialize unmarshals the Action
func (t *Action) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, t)
}

// GetSerializedOutputs returns the outputs in the Action serialized
func (t *Action) GetSerializedOutputs() ([][]byte, error) {
	res := make([][]byte, len(t.OutputTokens))
	var err error
	for i, token := range t.OutputTokens {
		res[i], err = token.Serialize()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// GetOutputCommitments returns the Pedersen commitments in the Action
func (t *Action) GetOutputCommitments() []*math.G1 {
	com := make([]*math.G1, len(t.OutputTokens))
	for i := 0; i < len(com); i++ {
		com[i] = t.OutputTokens[i].Data
	}
	return com
}

// IsGraphHiding returns false
// zkatdlog is not graph hiding
func (t *Action) IsGraphHiding() bool {
	return false
}

// GetMetadata returns metadata of the Action
func (t *Action) GetMetadata() map[string][]byte {
	return t.Metadata
}

func (t *Action) Validate() error {
	return nil
}

func (t *Action) ExtraSigners() []driver.Identity {
	return nil
}
