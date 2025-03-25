/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/protos"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ActionInput struct {
	ID    token2.ID
	Token []byte
}

func (i *ActionInput) ToProtos() (*actions.IssueActionInput, error) {
	return &actions.IssueActionInput{
		Id: &actions.TokenID{
			Id:    i.ID.TxId,
			Index: i.ID.Index,
		},
		Token: i.Token,
	}, nil
}

func (i *ActionInput) FromProtos(p *actions.IssueActionInput) error {
	if p.Id != nil {
		i.ID.TxId = p.Id.Id
		i.ID.Index = p.Id.Index
	}
	i.Token = p.Token
	return nil
}

// Action specifies an issue of one or more tokens
type Action struct {
	// Issuer is the identity of issuer
	Issuer driver.Identity
	// Inputs are the tokens to be redeemed by this issue action
	Inputs []*ActionInput
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
		if input == nil {
			res[i] = nil
			continue
		}
		res[i] = &input.ID
	}
	return res
}

func (i *Action) GetSerializedInputs() ([][]byte, error) {
	res := make([][]byte, len(i.Inputs))
	for i, input := range i.Inputs {
		if input == nil {
			res[i] = nil
			continue
		}
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
		if input == nil {
			return errors.Errorf("nil input in issue action")
		}
		if len(input.Token) == 0 {
			return errors.Errorf("nil input token in issue action")
		}
		if len(input.ID.TxId) == 0 {
			return errors.Errorf("nil input id in issue action")
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
	// inputs
	inputs, err := protos.ToProtosSlice[actions.IssueActionInput, *ActionInput](i.Inputs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize inputs")
	}

	// outputs
	outputs, err := protos.ToProtosSliceFunc(i.Outputs, func(output *token.Token) (*actions.IssueActionOutput, error) {
		data, err := utils.ToProtoG1(output.Data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to serialize output")
		}
		return &actions.IssueActionOutput{
			Token: &actions.Token{
				Owner: output.Owner,
				Data:  data,
			},
		}, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize outputs")
	}

	issueAction := &actions.IssueAction{
		Issuer: &pp.Identity{
			Raw: i.Issuer,
		},
		Inputs:  inputs,
		Outputs: outputs,
		Proof: &actions.Proof{
			Proof: i.Proof,
		},
		Metadata: i.Metadata,
	}
	return proto.Marshal(issueAction)
}

// Deserialize un-marshals IssueAction
func (i *Action) Deserialize(raw []byte) error {
	issueAction := &actions.IssueAction{}
	err := proto.Unmarshal(raw, issueAction)
	if err != nil {
		return errors.Wrap(err, "failed to deserialize issue action")
	}

	// inputs
	i.Inputs = make([]*ActionInput, len(issueAction.Inputs))
	i.Inputs = slices.GenericSliceOfPointers[ActionInput](len(issueAction.Inputs))
	if err := protos.FromProtosSlice(issueAction.Inputs, i.Inputs); err != nil {
		return errors.Wrap(err, "failed unmarshalling receivers metadata")
	}

	// outputs
	i.Outputs = make([]*token.Token, len(issueAction.Outputs))
	for j, output := range issueAction.Outputs {
		if output == nil || output.Token == nil {
			continue
		}
		data, err := utils.FromG1Proto(output.Token.Data)
		if err != nil {
			return errors.Wrapf(err, "failed to deserialize output")
		}
		i.Outputs[j] = &token.Token{
			Owner: output.Token.Owner,
			Data:  data,
		}
	}

	if issueAction.Proof != nil {
		i.Proof = issueAction.Proof.Proof
	}
	if issueAction.Issuer != nil {
		i.Issuer = issueAction.Issuer.Raw
	}
	i.Metadata = issueAction.Metadata

	return nil
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
