/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

// SigningIdentity signs TokenRequest
type SigningIdentity interface {
	driver.SigningIdentity
}

// Sender produces a signed TokenRequest
type Sender struct {
	// Signers is an array of Signer that matches the owners of the inputs
	// to be spent in the transfer action
	Signers []driver.Signer
	// Inputs to be spent in the transfer
	Inputs []*token.Token
	// InputIDs is the identifiers of the Inputs to be spent
	InputIDs []string
	// contains the opening of the inputs to be spent
	InputInformation []*token.Metadata
	// PublicParams refers to the public cryptographic parameters to be used
	// to produce the TokenRequest
	PublicParams *crypto.PublicParams
}

// NewSender returns a Sender
func NewSender(signers []driver.Signer, tokens []*token.Token, ids []string, inf []*token.Metadata, pp *crypto.PublicParams) (*Sender, error) {
	if len(signers) != len(tokens) || len(tokens) != len(inf) || len(ids) != len(inf) {
		return nil, errors.Errorf("number of tokens to be spent does not match number of opening")
	}
	return &Sender{Signers: signers, Inputs: tokens, InputIDs: ids, InputInformation: inf, PublicParams: pp}, nil
}

// GenerateZKTransfer produces a TransferAction and an array of Metadata
// that corresponds to the openings of the newly created outputs
func (s *Sender) GenerateZKTransfer(values []uint64, owners [][]byte) (*TransferAction, []*token.Metadata, error) {
	if len(values) != len(owners) {
		return nil, nil, errors.Errorf("cannot generate transfer: number of values [%d] does not match number of recipients [%d]", len(values), len(owners))
	}
	in := getTokenData(s.Inputs)
	intw := make([]*token.TokenDataWitness, len(s.InputInformation))
	for i := 0; i < len(s.InputInformation); i++ {
		if s.InputInformation[0].Type != s.InputInformation[i].Type {
			return nil, nil, errors.New("cannot generate transfer: please choose inputs of the same token type")
		}
		intw[i] = &token.TokenDataWitness{Value: s.InputInformation[i].Value, Type: s.InputInformation[i].Type, BlindingFactor: s.InputInformation[i].BlindingFactor}
	}
	out, outtw, err := token.GetTokensWithWitness(values, s.InputInformation[0].Type, s.PublicParams.PedParams, math.Curves[s.PublicParams.Curve])
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate transfer")
	}
	prover := NewProver(intw, outtw, in, out, s.PublicParams)
	proof, err := prover.Prove()
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate zero-knowledge proof for transfer")
	}

	transfer, err := NewTransfer(s.InputIDs, in, out, owners, proof)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to produce transfer action")
	}
	inf := make([]*token.Metadata, len(owners))
	for i := 0; i < len(inf); i++ {
		inf[i] = &token.Metadata{
			Type:           s.InputInformation[0].Type,
			Value:          outtw[i].Value,
			BlindingFactor: outtw[i].BlindingFactor,
			Owner:          owners[i],
		}
	}
	return transfer, inf, nil
}

// SignTokenActions produces a signature for each input spent by the Sender
func (s *Sender) SignTokenActions(raw []byte, txID string) ([][]byte, error) {
	//todo check token actions (is this still needed?)
	signatures := make([][]byte, len(s.Signers))
	var err error
	for i := 0; i < len(signatures); i++ {
		signatures[i], err = s.Signers[i].Sign(append(raw, []byte(txID)...))
		if err != nil {
			return nil, errors.Wrap(err, "failed to sign token requests")
		}
	}
	return signatures, nil
}

// TransferAction specifies a transfer of one or more tokens
type TransferAction struct {
	// Inputs specify the identifiers in of the tokens to be spent
	Inputs []string
	// InputCommitments are the PedersenCommitments in the inputs
	InputCommitments []*math.G1
	// OutputTokens are the new tokens resulting from the transfer
	OutputTokens []*token.Token
	// ZK Proof that shows that the transfer is correct
	Proof []byte
	// Metadata contains the transfer action's metadata
	Metadata map[string][]byte
}

// NewTransfer returns the TransferAction that matches the passed arguments
func NewTransfer(inputs []string, inputCommitments []*math.G1, outputs []*math.G1, owners [][]byte, proof []byte) (*TransferAction, error) {
	if len(outputs) != len(owners) {
		return nil, errors.Errorf("number of recipients [%d] does not match number of outputs [%d]", len(outputs), len(owners))
	}
	if len(inputs) != len(inputCommitments) {
		return nil, errors.Errorf("number of inputs [%d] does not match number of inputCommitments [%d]", len(inputs), len(inputCommitments))
	}
	tokens := make([]*token.Token, len(owners))
	for i, o := range outputs {
		tokens[i] = &token.Token{Data: o, Owner: owners[i]}
	}
	return &TransferAction{
		Inputs:           inputs,
		InputCommitments: inputCommitments,
		OutputTokens:     tokens,
		Proof:            proof,
		Metadata:         map[string][]byte{},
	}, nil
}

// GetInputs returns the inputs in the TransferAction
func (t *TransferAction) GetInputs() ([]string, error) {
	return t.Inputs, nil
}

// NumOutputs returns the number of outputs in the TransferAction
func (t *TransferAction) NumOutputs() int {
	return len(t.OutputTokens)
}

// GetOutputs returns the outputs in the TransferAction
func (t *TransferAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, outputToken := range t.OutputTokens {
		res = append(res, outputToken)
	}
	return res
}

// IsRedeemAt checks if output in the TransferAction at the passed index is redeemed
func (t *TransferAction) IsRedeemAt(index int) bool {
	return t.OutputTokens[index].IsRedeem()
}

// SerializeOutputAt marshals the output in the TransferAction at the passed index
func (t *TransferAction) SerializeOutputAt(index int) ([]byte, error) {
	return t.OutputTokens[index].Serialize()
}

// Serialize marshals the TransferAction
func (t *TransferAction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// GetProof returns the proof in the TransferAction
func (t *TransferAction) GetProof() []byte {
	return t.Proof
}

// Deserialize unmarshals the TransferAction
func (t *TransferAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, t)
}

// GetSerializedOutputs returns the outputs in the TransferAction serialized
func (t *TransferAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for _, token := range t.OutputTokens {
		r, err := token.Serialize()
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// GetOutputCommitments returns the Pedersen commitments in the TransferAction
func (t *TransferAction) GetOutputCommitments() []*math.G1 {
	com := make([]*math.G1, len(t.OutputTokens))
	for i := 0; i < len(com); i++ {
		com[i] = t.OutputTokens[i].Data
	}
	return com
}

// IsGraphHiding returns false
// zkatdlog is not graph hiding
func (t *TransferAction) IsGraphHiding() bool {
	return false
}

// GetMetadata returns metadata of the TransferAction
func (t *TransferAction) GetMetadata() map[string][]byte {
	return t.Metadata
}

func getTokenData(tokens []*token.Token) []*math.G1 {
	tokenData := make([]*math.G1, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tokenData[i] = tokens[i].Data
	}
	return tokenData
}
