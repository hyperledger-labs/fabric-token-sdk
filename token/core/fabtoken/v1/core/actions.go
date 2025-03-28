/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// OutputMetadata contains a serialization of the issuer of the token.
// type, value and owner of token can be derived from the token itself.
type OutputMetadata fabtoken.Metadata

// Deserialize un-marshals Metadata
func (m *OutputMetadata) Deserialize(b []byte) error {
	typed, err := fabtoken.UnmarshalTypedToken(b)
	if err != nil {
		return errors.Wrapf(err, "failed deserializing metadata")
	}
	return json.Unmarshal(typed.Token, m)
}

// Serialize un-marshals Metadata
func (m *OutputMetadata) Serialize() ([]byte, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, errors.Wrapf(err, "failed serializing token")
	}
	return fabtoken.WrapMetadataWithType(raw)
}

// Output carries the output of an action
type Output fabtoken.Token

// Serialize marshals a Token
func (t *Output) Serialize() ([]byte, error) {
	raw, err := json.Marshal(t)
	if err != nil {
		return nil, errors.Wrapf(err, "failed serializing token")
	}
	return fabtoken.WrapTokenWithType(raw)
}

// Deserialize unmarshals Token
func (t *Output) Deserialize(bytes []byte) error {
	typed, err := fabtoken.UnmarshalTypedToken(bytes)
	if err != nil {
		return errors.Wrapf(err, "failed deserializing token")
	}
	return json.Unmarshal(typed.Token, t)
}

// IsRedeem returns true if the owner of a Token is empty
// todo update interface to account for nil t.Token.Owner and nil t.Token
func (t *Output) IsRedeem() bool {
	return len(t.Owner) == 0
}

func (t *Output) GetOwner() []byte {
	return t.Owner
}

// IssueAction encodes a fabtoken Issue
type IssueAction struct {
	// issuer's public key
	Issuer driver.Identity
	// new tokens to be issued
	Outputs []*Output
	// metadata of the issue action
	Metadata map[string][]byte
}

func (i *IssueAction) NumInputs() int {
	return 0
}

func (i *IssueAction) GetInputs() []*token.ID {
	return nil
}

func (i *IssueAction) GetSerializedInputs() ([][]byte, error) {
	return nil, nil
}

func (i *IssueAction) GetSerialNumbers() []string {
	return nil
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

// IsGraphHiding returns false, indicating that fabtoken does not hide the transaction graph
func (i *IssueAction) IsGraphHiding() bool {
	return false
}

func (i *IssueAction) Validate() error {
	if i.Issuer.IsNone() {
		return errors.Errorf("issuer is not set")
	}
	if len(i.Outputs) == 0 {
		return errors.Errorf("no outputs in issue action")
	}
	for _, output := range i.Outputs {
		if output == nil {
			return errors.Errorf("nil output in issue action")
		}
	}
	return nil
}

func (i *IssueAction) ExtraSigners() []driver.Identity {
	return nil
}

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
