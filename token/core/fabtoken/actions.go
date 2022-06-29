/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	"encoding/json"
	"github.com/pkg/errors"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// TokenInformation contains a serialization of the issuer of the token.
// type, value and owner of token can be derived from the token itself.
type TokenInformation struct {
	Issuer []byte
}

// Deserialize un-marshals TokenInformation
func (inf *TokenInformation) Deserialize(b []byte) error {
	return json.Unmarshal(b, inf)
}

// Serialize marshals TokenInformation
func (inf *TokenInformation) Serialize() ([]byte, error) {
	return json.Marshal(inf)
}

// TransferOutput carries the output of a TransferAction
type TransferOutput struct {
	Output *token2.Token
}

// Serialize marshals a TransferOutput
func (t *TransferOutput) Serialize() ([]byte, error) {
	return json.Marshal(t.Output)
}

// IsRedeem returns true if the owner of a TransferOutput is empty
func (t *TransferOutput) IsRedeem() bool {
	return len(t.Output.Owner.Raw) == 0
}

// IssueAction encodes a fabtoken Issue
type IssueAction struct {
	// issuer's public key
	Issuer view.Identity
	// new tokens to be issued
	Outputs []*TransferOutput
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

// GetMetadata returns nil, indicating that IssueAction in fabtoken carries no metadata
func (i *IssueAction) GetMetadata() []byte {
	return nil
}

// TransferAction encodes a fabtoken transfer
type TransferAction struct {
	// identifier of token to be transferred
	Inputs []string
	// outputs to be created as a result of the transfer
	Outputs []*TransferOutput
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
func (t *TransferAction) IsRedeemAt(index int) bool {
	return t.Outputs[index].IsRedeem()
}

// IsGraphHiding returns false, indicating that fabtoken does not hide the transaction graph
func (t *TransferAction) IsGraphHiding() bool {
	return false
}

// SerializeOutputAt marshals the output at the specified index in TransferAction
func (t *TransferAction) SerializeOutputAt(index int) ([]byte, error) {
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

// GetMetadata returns nil, indicating that fabtoken TransferAction carries no metadata
func (t *TransferAction) GetMetadata() []byte {
	return nil
}
