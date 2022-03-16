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

// signing identity
type SigningIdentity interface {
	driver.SigningIdentity
}

type Sender struct {
	Signers          []driver.Signer
	Inputs           []*token.Token
	InputIDs         []string
	InputInformation []*token.TokenInformation // contains the opening of the inputs to be spent
	PublicParams     *crypto.PublicParams
}

func NewSender(signers []driver.Signer, tokens []*token.Token, ids []string, inf []*token.TokenInformation, pp *crypto.PublicParams) (*Sender, error) {
	if len(signers) != len(tokens) || len(tokens) != len(inf) {
		return nil, errors.Errorf("number of tokens to be spent does not match number of opening")
	}
	return &Sender{Signers: signers, Inputs: tokens, InputIDs: ids, InputInformation: inf, PublicParams: pp}, nil
}

func (s *Sender) GenerateZKTransfer(values []uint64, owners [][]byte) (*TransferAction, []*token.TokenInformation, error) {
	out, outtw, err := token.GetTokensWithWitness(values, s.InputInformation[0].Type, s.PublicParams.ZKATPedParams, math.Curves[s.PublicParams.Curve])
	if err != nil {
		return nil, nil, err
	}

	in := getTokenData(s.Inputs)
	intw := make([]*token.TokenDataWitness, len(s.InputInformation))
	for i := 0; i < len(s.InputInformation); i++ {
		intw[i] = &token.TokenDataWitness{Value: s.InputInformation[i].Value, Type: s.InputInformation[i].Type, BlindingFactor: s.InputInformation[i].BlindingFactor}
	}
	prover := NewProver(intw, outtw, in, out, s.PublicParams)
	proof, err := prover.Prove()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate zero-knowledge proof for transfer request")
	}

	transfer, err := NewTransfer(s.InputIDs, in, out, owners, proof)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate transfer")
	}
	inf := make([]*token.TokenInformation, len(owners))
	for i := 0; i < len(inf); i++ {
		inf[i] = &token.TokenInformation{
			Type:           s.InputInformation[0].Type,
			Value:          outtw[i].Value,
			BlindingFactor: outtw[i].BlindingFactor,
			Owner:          owners[i],
		}
	}
	return transfer, inf, nil
}

func (s *Sender) SignTokenActions(raw []byte, txID string) ([][]byte, error) {
	//todo check token actions
	signatures := make([][]byte, len(s.Signers))
	var err error
	for i := 0; i < len(signatures); i++ {
		signatures[i], err = s.Signers[i].Sign(append(raw, []byte(txID)...))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to sign token requests")
		}
	}
	return signatures, nil
}

// Transfer specifies a transfer of one or more tokens
type TransferAction struct {
	// Inputs specify the identifiers in the rwset of the tokens to be transferred
	Inputs           []string
	InputCommitments []*math.G1
	// OutputTokens are the new tokens resulting from the transfer
	OutputTokens []*token.Token
	// ZK Proof
	Proof []byte
}

func NewTransfer(inputs []string, inputCommitments []*math.G1, outputs []*math.G1, owners [][]byte, proof []byte) (*TransferAction, error) {
	if len(outputs) != len(owners) {
		return nil, errors.Errorf("number of owners does not match number of tokens")
	}
	tokens := make([]*token.Token, len(owners))
	for i, o := range outputs {
		tokens[i] = &token.Token{Data: o, Owner: owners[i]}
	}
	return &TransferAction{
		Inputs:           inputs,
		InputCommitments: inputCommitments,
		OutputTokens:     tokens,
		Proof:            proof}, nil
}

func (t *TransferAction) GetInputs() ([]string, error) {
	return t.Inputs, nil
}

func (t *TransferAction) NumOutputs() int {
	return len(t.OutputTokens)
}

func (t *TransferAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, outputToken := range t.OutputTokens {
		res = append(res, outputToken)
	}
	return res
}

func (t *TransferAction) IsRedeemAt(index int) bool {
	return t.OutputTokens[index].IsRedeem()
}

func (t *TransferAction) SerializeOutputAt(index int) ([]byte, error) {
	return t.OutputTokens[index].Serialize()
}

func (t *TransferAction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TransferAction) GetProof() []byte {
	return t.Proof
}

func (t *TransferAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, t)
}

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

func (t *TransferAction) GetOutputCommitments() []*math.G1 {
	com := make([]*math.G1, len(t.OutputTokens))
	for i := 0; i < len(com); i++ {
		com[i] = t.OutputTokens[i].Data
	}
	return com
}

func (t *TransferAction) IsGraphHiding() bool {
	return false
}

func (t *TransferAction) GetMetadata() []byte {
	return nil
}

func getTokenData(tokens []*token.Token) []*math.G1 {
	tokenData := make([]*math.G1, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tokenData[i] = tokens[i].Data
	}
	return tokenData
}
