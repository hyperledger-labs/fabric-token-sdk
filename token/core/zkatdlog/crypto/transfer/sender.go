/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"context"
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
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
	InputIDs []*token2.ID
	// contains the opening of the inputs to be spent
	InputInformation []*token.Metadata
	// PublicParams refers to the public cryptographic parameters to be used
	// to produce the TokenRequest
	PublicParams *crypto.PublicParams
}

// NewSender returns a Sender
func NewSender(signers []driver.Signer, tokens []*token.Token, ids []*token2.ID, inf []*token.Metadata, pp *crypto.PublicParams) (*Sender, error) {
	if (signers != nil && len(signers) != len(tokens)) || len(tokens) != len(inf) || len(ids) != len(inf) {
		return nil, errors.Errorf("number of tokens to be spent does not match number of opening")
	}
	return &Sender{Signers: signers, Inputs: tokens, InputIDs: ids, InputInformation: inf, PublicParams: pp}, nil
}

// GenerateZKTransfer produces a Action and an array of ValidationRecords
// that corresponds to the openings of the newly created outputs
func (s *Sender) GenerateZKTransfer(ctx context.Context, values []uint64, owners [][]byte) (*Action, []*token.Metadata, error) {
	span := trace.SpanFromContext(ctx)
	if len(values) != len(owners) {
		return nil, nil, errors.Errorf("cannot generate transfer: number of values [%d] does not match number of recipients [%d]", len(values), len(owners))
	}
	span.AddEvent("get_token_data")
	in := getTokenData(s.Inputs)
	intw := make([]*token.TokenDataWitness, len(s.InputInformation))
	for i := 0; i < len(s.InputInformation); i++ {
		if s.InputInformation[0].Type != s.InputInformation[i].Type {
			return nil, nil, errors.New("cannot generate transfer: please choose inputs of the same token type")
		}
		v, err := s.InputInformation[i].Value.Uint()
		if err != nil {
			return nil, nil, errors.New("cannot generate transfer: invalid value")
		}

		intw[i] = &token.TokenDataWitness{
			Value:          v,
			Type:           s.InputInformation[i].Type,
			BlindingFactor: s.InputInformation[i].BlindingFactor,
		}
	}
	span.AddEvent("get_tokens_with_witness")
	out, outtw, err := token.GetTokensWithWitness(values, s.InputInformation[0].Type, s.PublicParams.PedersenGenerators, math.Curves[s.PublicParams.Curve])
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate transfer")
	}
	span.AddEvent("create_new_prover")
	prover, err := NewProver(intw, outtw, in, out, s.PublicParams)
	if err != nil {
		return nil, nil, errors.New("cannot generate transfer")
	}
	span.AddEvent("prove")
	proof, err := prover.Prove()
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate zero-knowledge proof for transfer")
	}

	span.AddEvent("create_new_transfer")
	transfer, err := NewTransfer(s.InputIDs, s.Inputs, out, owners, proof)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to produce transfer action")
	}
	inf := make([]*token.Metadata, len(owners))
	for i := 0; i < len(inf); i++ {
		inf[i] = &token.Metadata{
			Type:           s.InputInformation[0].Type,
			Value:          math.Curves[s.PublicParams.Curve].NewZrFromUint64(outtw[i].Value),
			BlindingFactor: outtw[i].BlindingFactor,
			Owner:          owners[i],
		}
	}
	return transfer, inf, nil
}

// SignTokenActions produces a signature for each input spent by the Sender
func (s *Sender) SignTokenActions(raw []byte, txID string) ([][]byte, error) {
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

// Action specifies a transfer of one or more tokens
type Action struct {
	// Inputs specify the identifiers in of the tokens to be spent
	Inputs []*token2.ID
	// InputCommitments are the PedersenCommitments in the inputs
	InputTokens []*token.Token
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
	for _, token := range t.InputTokens {
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

func getTokenData(tokens []*token.Token) []*math.G1 {
	tokenData := make([]*math.G1, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tokenData[i] = tokens[i].Data
	}
	return tokenData
}
