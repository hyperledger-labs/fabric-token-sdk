/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package actions

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// TransferAction encodes a fabtoken transfer
type TransferAction struct {
	// identifier of token to be transferred
	Inputs []*token.ID
	// InputTokens are the inputs transferred by this action
	InputTokens []*Output
	// outputs to be created as a result of the transfer
	Outputs []*Output
	// Metadata contains the transfer action's metadata
	Metadata map[string][]byte
}

func (t *TransferAction) NumInputs() int {
	return len(t.Inputs)
}

// Serialize marshals TransferAction
func (t *TransferAction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// NumOutputs returns the number of outputs in an TransferAction
func (t *TransferAction) NumOutputs() int {
	return len(t.Outputs)
}

// GetSerializedOutputs returns the serialization of the outputs in a TransferAction
func (t *TransferAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for k, output := range t.Outputs {
		if output == nil {
			return nil, errors.Errorf("cannot serialize transfer action outputs: nil output at index [%d]", k)
		}
		ser, err := output.Serialize()
		if err != nil {
			return nil, errors.Errorf("failed to serialize output [%d] in transfer action", k)
		}
		res = append(res, ser)
	}
	return res, nil
}

// GetOutputs returns the outputs in a TransferAction
func (t *TransferAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, output := range t.Outputs {
		res = append(res, output)
	}
	return res
}

// IsRedeemAt returns true if the output at the specified index is a redeemed output
// todo update interface to account for nil t.outputs[index]
func (t *TransferAction) IsRedeemAt(index int) bool {
	return t.Outputs[index].IsRedeem()
}

// IsGraphHiding returns false, indicating that fabtoken does not hide the transaction graph
func (t *TransferAction) IsGraphHiding() bool {
	return false
}

// SerializeOutputAt marshals the output at the specified index in TransferAction
func (t *TransferAction) SerializeOutputAt(index int) ([]byte, error) {
	if index >= len(t.Outputs) {
		return nil, errors.Errorf("failed to serialize output in transfer action: it does not exist")
	}
	if t.Outputs[index] == nil {
		return nil, errors.Errorf("failed to serialize output in transfer action: nil output at index [%d]", index)
	}
	return t.Outputs[index].Serialize()
}

// GetInputs returns inputs of the TransferAction
func (t *TransferAction) GetInputs() []*token.ID {
	return t.Inputs
}

func (t *TransferAction) GetSerializedInputs() ([][]byte, error) {
	var res [][]byte
	for _, token := range t.InputTokens {
		r, err := token.Serialize()
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (t *TransferAction) GetSerialNumbers() []string {
	return nil
}

// Deserialize un-marshals TransferAction
func (t *TransferAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, t)
}

// GetMetadata returns the transfer action's metadata
func (t *TransferAction) GetMetadata() map[string][]byte {
	return t.Metadata
}

func (t *TransferAction) Validate() error {
	return nil
}

func (t *TransferAction) ExtraSigners() []driver.Identity {
	return nil
}
