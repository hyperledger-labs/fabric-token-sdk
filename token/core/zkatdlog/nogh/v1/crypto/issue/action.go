/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ActionInput struct {
	ID    token2.ID
	Token []byte
}

// Action specifies an issue of one or more tokens
type Action struct {
	// Issuer is the identity of issuer
	Issuer driver.Identity
	// Inputs are the tokens to be redeemed by this issue action
	Inputs []ActionInput
	// Outputs are the newly issued tokens
	Outputs []*token.Token `protobuf:"bytes,1,rep,name=outputs,proto3" json:"outputs,omitempty"`
	// Proof carries the ZKP of IssueAction validity
	Proof []byte
	// Metadata of the issue action
	Metadata map[string][]byte
}

// NewAction instantiates an IssueAction given the passed arguments
func NewAction(issuer []byte, coms []*math.G1, owners [][]byte, proof []byte) (*Action, error) {
	if len(owners) != len(coms) {
		return nil, errors.New("number of owners does not match number of tokens")
	}

	outputs := make([]*token.Token, len(coms))
	for i, c := range coms {
		outputs[i] = &token.Token{Owner: owners[i], Data: c}
	}

	return &Action{
		Issuer:  issuer,
		Outputs: outputs,
		Proof:   proof,
	}, nil
}

func (i *Action) NumInputs() int {
	return len(i.Inputs)
}

func (i *Action) GetInputs() []*token2.ID {
	res := make([]*token2.ID, len(i.Inputs))
	for i, input := range i.Inputs {
		res[i] = &input.ID
	}
	return res
}

func (i *Action) GetSerializedInputs() ([][]byte, error) {
	res := make([][]byte, len(i.Inputs))
	for i, input := range i.Inputs {
		res[i] = input.Token
	}
	return res, nil
}

func (i *Action) GetSerialNumbers() []string {
	return nil
}

// GetMetadata returns IssueAction metadata if there is any.
func (i *Action) GetMetadata() map[string][]byte {
	return i.Metadata
}

// IsAnonymous returns a Boolean. True if IssueAction is anonymous, and False otherwise.
func (i *Action) IsAnonymous() bool {
	return false
}

// NumOutputs returns the number of outputs in IssueAction
func (i *Action) NumOutputs() int {
	return len(i.Outputs)
}

// GetOutputs returns the Outputs in IssueAction
func (i *Action) GetOutputs() []driver.Output {
	res := make([]driver.Output, len(i.Outputs))
	for i, tok := range i.Outputs {
		res[i] = tok
	}
	return res
}

// GetSerializedOutputs returns the serialization of Outputs
func (i *Action) GetSerializedOutputs() ([][]byte, error) {
	res := make([][]byte, len(i.Outputs))
	for i, tok := range i.Outputs {
		if tok == nil {
			return nil, errors.New("invalid issue: there is a nil output")
		}
		var err error
		res[i], err = tok.Serialize()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// GetIssuer returns the Issuer of IssueAction
func (i *Action) GetIssuer() []byte {
	return i.Issuer
}

// IsGraphHiding returns false, this driver does not hide the transaction graph
func (i *Action) IsGraphHiding() bool {
	return false
}

func (i *Action) Validate() error {
	if i.Issuer.IsNone() {
		return errors.Errorf("issuer is not set")
	}
	for _, input := range i.Inputs {
		if len(input.Token) == 0 {
			return errors.Errorf("nil input token in issue action")
		}
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

func (i *Action) ExtraSigners() []driver.Identity {
	return nil
}

// Serialize marshal IssueAction
func (i *Action) Serialize() ([]byte, error) {
	return json.Marshal(i)
}

// Deserialize un-marshals IssueAction
func (i *Action) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, i)
}

// GetCommitments return the Pedersen commitment of (type, value) in the Outputs
func (i *Action) GetCommitments() ([]*math.G1, error) {
	com := make([]*math.G1, len(i.Outputs))
	for j := 0; j < len(com); j++ {
		if i.Outputs[j] == nil {
			return nil, errors.New("invalid issue: there is a nil output")
		}
		com[j] = i.Outputs[j].Data
	}
	return com, nil
}

// GetProof returns IssueAction ZKP
func (i *Action) GetProof() []byte {
	return i.Proof
}
