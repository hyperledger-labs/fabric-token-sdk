/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	factions "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/protos-go/actions"
	fv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/protos"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const ProtocolV1 = 1

type ActionInput struct {
	ID             *token2.ID
	Token          *token.Token
	UpgradeWitness *token.UpgradeWitness
}

func (a *ActionInput) ToProtos() (*actions.TransferActionInput, error) {
	var id *actions.TokenID
	if a.ID != nil {
		id = &actions.TokenID{
			Id:    a.ID.TxId,
			Index: a.ID.Index,
		}
	}
	var input *actions.Token
	if a.Token != nil {
		data, err := utils.ToProtoG1(a.Token.Data)
		if err != nil {
			return nil, err
		}
		input = &actions.Token{
			Owner: a.Token.Owner,
			Data:  data,
		}
	}
	var witness *actions.TransferActionInputUpgradeWitness
	if a.UpgradeWitness != nil {
		blindingFactor, err := utils.ToProtoZr(a.UpgradeWitness.BlindingFactor)
		if err != nil {
			return nil, err
		}
		var fabtoken *factions.Token
		if a.UpgradeWitness.FabToken != nil {
			fabtoken = &factions.Token{
				Owner:    a.UpgradeWitness.FabToken.Owner,
				Type:     string(a.UpgradeWitness.FabToken.Type),
				Quantity: a.UpgradeWitness.FabToken.Quantity,
			}
		}
		witness = &actions.TransferActionInputUpgradeWitness{
			Output:         fabtoken,
			BlindingFactor: blindingFactor,
		}
	}

	return &actions.TransferActionInput{
		TokenId:        id,
		Input:          input,
		UpgradeWitness: witness,
	}, nil
}

func (a *ActionInput) FromProtos(input *actions.TransferActionInput) error {
	if input.TokenId != nil {
		a.ID = &token2.ID{
			TxId:  input.TokenId.Id,
			Index: input.TokenId.Index,
		}
	}
	if input.Input != nil {
		data, err := utils.FromG1Proto(input.Input.Data)
		if err != nil {
			return err
		}
		a.Token = &token.Token{
			Owner: input.Input.Owner,
			Data:  data,
		}
	}
	if input.UpgradeWitness != nil {
		blindingFactor, err := utils.FromZrProto(input.UpgradeWitness.BlindingFactor)
		if err != nil {
			return err
		}
		var fabtoken *fv1.Output
		if input.UpgradeWitness.Output != nil {
			fabtoken = &fv1.Output{
				Owner:    input.UpgradeWitness.Output.Owner,
				Type:     token2.Type(input.UpgradeWitness.Output.Type),
				Quantity: input.UpgradeWitness.Output.Quantity,
			}
		}
		a.UpgradeWitness = &token.UpgradeWitness{
			FabToken:       fabtoken,
			BlindingFactor: blindingFactor,
		}
	}
	return nil
}

// Action specifies a transfer of one or more tokens
type Action struct {
	// Inputs specify the identifiers in of the tokens to be spent
	Inputs []*ActionInput
	// Outputs are the new tokens resulting from the transfer
	Outputs []*token.Token
	// ZK Proof that shows that the transfer is correct
	Proof []byte
	// Metadata contains the transfer action's metadata
	Metadata map[string][]byte
	// Issuer contains the identity of the issuer to sign the transfer action
	Issuer driver.Identity
}

// NewTransfer returns the Action that matches the passed arguments
func NewTransfer(tokenIDs []*token2.ID, inputToken []*token.Token, commitments []*math.G1, owners [][]byte, proof []byte) (*Action, error) {
	if len(commitments) != len(owners) {
		return nil, errors.Errorf("number of recipients [%d] does not match number of outputs [%d]", len(commitments), len(owners))
	}
	if len(tokenIDs) != len(inputToken) {
		return nil, errors.Errorf("number of inputs [%d] does not match number of input tokens [%d]", len(tokenIDs), len(inputToken))
	}

	inputs := make([]*ActionInput, len(tokenIDs))
	for i, id := range tokenIDs {
		inputs[i] = &ActionInput{
			ID:    id,
			Token: inputToken[i],
		}
	}

	tokens := make([]*token.Token, len(owners))
	for i, o := range commitments {
		tokens[i] = &token.Token{Data: o, Owner: owners[i]}
	}
	return &Action{
		Inputs:   inputs,
		Outputs:  tokens,
		Proof:    proof,
		Metadata: map[string][]byte{},
		Issuer:   nil,
	}, nil
}

func (t *Action) NumInputs() int {
	return len(t.Inputs)
}

// GetInputs returns the inputs in the Action
func (t *Action) GetInputs() []*token2.ID {
	res := make([]*token2.ID, len(t.Inputs))
	for i, input := range t.Inputs {
		res[i] = input.ID
	}
	return res
}

func (t *Action) GetSerializedInputs() ([][]byte, error) {
	var res [][]byte
	for _, input := range t.Inputs {
		if input == nil {
			res = append(res, nil)
			continue
		}
		if w := input.UpgradeWitness; w != nil {
			ser, err := w.FabToken.Serialize()
			if err != nil {
				return nil, err
			}
			res = append(res, ser)
			continue
		}
		r, err := input.Token.Serialize()
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
	return len(t.Outputs)
}

// GetOutputs returns the outputs in the Action
func (t *Action) GetOutputs() []driver.Output {
	res := make([]driver.Output, len(t.Outputs))
	for i, outputToken := range t.Outputs {
		res[i] = outputToken
	}
	return res
}

// IsRedeemAt checks if output in the Action at the passed index is redeemed
func (t *Action) IsRedeemAt(index int) bool {
	return t.Outputs[index].IsRedeem()
}

// SerializeOutputAt marshals the output in the Action at the passed index
func (t *Action) SerializeOutputAt(index int) ([]byte, error) {
	return t.Outputs[index].Serialize()
}

// GetSerializedOutputs returns the outputs in the Action serialized
func (t *Action) GetSerializedOutputs() ([][]byte, error) {
	res := make([][]byte, len(t.Outputs))
	var err error
	for i, token := range t.Outputs {
		res[i], err = token.Serialize()
		if err != nil {
			return nil, err
		}
	}
	return res, nil
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

// GetIssuer returns the issuer to sign the transaction
func (t *Action) GetIssuer() driver.Identity {
	return t.Issuer
}

func (t *Action) Validate() error {
	if len(t.Inputs) == 0 {
		return errors.Errorf("invalid number of token inputs, expected at least 1")
	}
	for i, in := range t.Inputs {
		if in == nil {
			return errors.Errorf("invalid input at index [%d], empty input", i)
		}
		if in.ID == nil {
			return errors.Errorf("invalid input's ID at index [%d], it is empty", i)
		}
		if len(in.ID.TxId) == 0 {
			return errors.Errorf("invalid input's ID at index [%d], tx id is empty", i)
		}
		if in.Token == nil {
			return errors.Errorf("invalid input's token at index [%d], empty token", i)
		}
		if err := in.Token.Validate(true); err != nil {
			return errors.Wrapf(err, "invalid input token at index [%d]", i)
		}

		if in.UpgradeWitness != nil {
			if err := in.UpgradeWitness.Validate(); err != nil {
				return errors.Wrapf(err, "invalid input's upgrade witness at index [%d]", i)
			}
		}
	}
	if len(t.Outputs) == 0 {
		return errors.Errorf("invalid number of token outputs, expected at least 1")
	}
	for i, out := range t.Outputs {
		if out == nil {
			return errors.Errorf("invalid output token at index [%d]", i)
		}
		if err := out.Validate(false); err != nil {
			return errors.Wrapf(err, "invalid output at index [%d]", i)
		}
	}
	return nil
}

func (t *Action) ExtraSigners() []driver.Identity {
	return nil
}

// Serialize marshal TransferAction
func (t *Action) Serialize() ([]byte, error) {
	// inputs
	inputs, err := protos.ToProtosSlice[actions.TransferActionInput, *ActionInput](t.Inputs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize inputs")
	}

	// outputs
	outputs, err := protos.ToProtosSliceFunc(t.Outputs, func(output *token.Token) (*actions.TransferActionOutput, error) {
		data, err := utils.ToProtoG1(output.Data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to serialize output")
		}
		return &actions.TransferActionOutput{
			Token: &actions.Token{
				Owner: output.Owner,
				Data:  data,
			},
		}, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize outputs")
	}

	var issuer *pp.Identity
	if t.Issuer != nil {
		issuer = &pp.Identity{
			Raw: t.Issuer.Bytes(),
		}
	}

	action := &actions.TransferAction{
		Version: ProtocolV1,
		Inputs:  inputs,
		Outputs: outputs,
		Proof: &actions.Proof{
			Proof: t.Proof,
		},
		Metadata: t.Metadata,
		Issuer:   issuer,
	}
	return proto.Marshal(action)
}

// Deserialize un-marshals TransferAction
func (t *Action) Deserialize(raw []byte) error {
	action := &actions.TransferAction{}
	err := proto.Unmarshal(raw, action)
	if err != nil {
		return errors.Wrap(err, "failed to deserialize issue action")
	}

	// assert version
	if action.Version != ProtocolV1 {
		return errors.Errorf("invalid issue version, expected [%d], got [%d]", ProtocolV1, action.Version)
	}

	// inputs
	t.Inputs = make([]*ActionInput, len(action.Inputs))
	t.Inputs = slices.GenericSliceOfPointers[ActionInput](len(action.Inputs))
	if err := protos.FromProtosSlice(action.Inputs, t.Inputs); err != nil {
		return errors.Wrap(err, "failed unmarshalling receivers metadata")
	}

	// outputs
	t.Outputs = make([]*token.Token, len(action.Outputs))
	for j, output := range action.Outputs {
		if output == nil || output.Token == nil {
			continue
		}
		data, err := utils.FromG1Proto(output.Token.Data)
		if err != nil {
			return errors.Wrapf(err, "failed to deserialize output")
		}
		t.Outputs[j] = &token.Token{
			Owner: output.Token.Owner,
			Data:  data,
		}
	}

	if action.Proof != nil {
		t.Proof = action.Proof.Proof
	}
	t.Metadata = action.Metadata

	if action.Issuer != nil {
		t.Issuer = driver.Identity(action.Issuer.Raw)
	} else {
		t.Issuer = nil
	}

	return nil
}

// GetProof returns the proof in the Action
func (t *Action) GetProof() []byte {
	return t.Proof
}

// GetOutputCommitments returns the Pedersen commitments in the Action
func (t *Action) GetOutputCommitments() []*math.G1 {
	com := make([]*math.G1, len(t.Outputs))
	for i := 0; i < len(com); i++ {
		com[i] = t.Outputs[i].Data
	}
	return com
}

func (t *Action) InputTokens() []*token.Token {
	tokens := make([]*token.Token, len(t.Inputs))
	for i, in := range t.Inputs {
		tokens[i] = in.Token
	}
	return tokens
}
