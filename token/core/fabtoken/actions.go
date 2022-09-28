/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// OutputMetadata contains a serialization of the issuer of the token.
// type, value and owner of token can be derived from the token itself.
type OutputMetadata struct {
	Issuer []byte
}

// Deserialize un-marshals OutputMetadata
func (m *OutputMetadata) Deserialize(b []byte) error {
	return json.Unmarshal(b, m)
}

// Serialize marshals OutputMetadata
func (m *OutputMetadata) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

// Output carries the output of an action
type Output struct {
	Output *token.Token
}

// Serialize marshals a Output
func (t *Output) Serialize() ([]byte, error) {
	return json.Marshal(t.Output)
}

// IsRedeem returns true if the owner of a Output is empty
// todo update interface to account for nil t.Output.Owner and nil t.Output
func (t *Output) IsRedeem() bool {
	return len(t.Output.Owner.Raw) == 0
}

// IssueAction encodes a fabtoken Issue
type IssueAction struct {
	// issuer's public key
	Issuer view.Identity
	// new tokens to be issued
	Outputs []*Output
	// metadata of the issue action
	Metadata map[string][]byte
}

// Serialize marshals IssueAction
func (i *IssueAction) Serialize() ([]byte, error) {
	return json.Marshal(i)
}

// Deserialize un-marshals IssueAction
func (i *IssueAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, i)
}

// NumOutputs returns the number of outputs in an IssueAction
func (i *IssueAction) NumOutputs() int {
	return len(i.Outputs)
}

// GetSerializedOutputs returns the serialization of the outputs in an IssueAction
func (i *IssueAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for k, output := range i.Outputs {
		if output == nil {
			return nil, errors.Errorf("cannot serialize issue action outputs: nil output at index [%d]", k)
		}
		ser, err := output.Serialize()
		if err != nil {
			return nil, errors.Errorf("failed to serialize output [%d] in issue action", k)
		}
		res = append(res, ser)
	}
	return res, nil
}

// GetOutputs returns the outputs in an IssueAction
func (i *IssueAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, output := range i.Outputs {
		res = append(res, output)
	}
	return res
}

// IsAnonymous returns false, indicating that the identity of issuers in fabtoken
// is revealed during issue
func (i *IssueAction) IsAnonymous() bool {
	return false
}

// GetIssuer returns the issuer encoded in IssueAction
func (i *IssueAction) GetIssuer() []byte {
	return i.Issuer
}

// GetMetadata returns the IssueAction metadata
func (i *IssueAction) GetMetadata() map[string][]byte {
	return i.Metadata
}

// TransferAction encodes a fabtoken transfer
type TransferAction struct {
	// identifier of token to be transferred
	Inputs []string
	// outputs to be created as a result of the transfer
	Outputs []*Output
	// Metadata contains the transfer action's metadata
	Metadata map[string][]byte
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
func (t *TransferAction) GetInputs() ([]string, error) {
	return t.Inputs, nil
}

// Deserialize un-marshals TransferAction
func (t *TransferAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, t)
}

// GetMetadata returns the transfer action's metadata
func (t *TransferAction) GetMetadata() map[string][]byte {
	return t.Metadata
}

// UnmarshalIssueTransferActions returns the deserialized issue and transfer actions contained in the passed TokenRequest
func UnmarshalIssueTransferActions(tr *driver.TokenRequest, binding string) ([]*IssueAction, []*TransferAction, error) {
	ia, err := UnmarshalIssueActions(tr.Issues)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to retrieve issue actions [%s]", binding)
	}
	ta, err := UnmarshalTransferActions(tr.Transfers)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to retrieve transfer actions [%s]", binding)
	}
	return ia, ta, nil
}

// UnmarshalTransferActions returns an array of deserialized TransferAction from raw bytes
func UnmarshalTransferActions(raw [][]byte) ([]*TransferAction, error) {
	res := make([]*TransferAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ta := &TransferAction{}
		if err := ta.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ta
	}
	return res, nil
}

// UnmarshalIssueActions returns an array of deserialized IssueAction from raw bytes
func UnmarshalIssueActions(raw [][]byte) ([]*IssueAction, error) {
	res := make([]*IssueAction, len(raw))
	for i := 0; i < len(raw); i++ {
		ia := &IssueAction{}
		if err := ia.Deserialize(raw[i]); err != nil {
			return nil, err
		}
		res[i] = ia
	}
	return res, nil
}
