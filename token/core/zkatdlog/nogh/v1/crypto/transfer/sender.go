/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"context"

	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
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
	PublicParams *v1.PublicParams
}

// NewSender returns a Sender
func NewSender(signers []driver.Signer, tokens []*token.Token, ids []*token2.ID, inf []*token.Metadata, pp *v1.PublicParams) (*Sender, error) {
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
		}
	}
	return transfer, inf, nil
}

// SignTokenActions produces a signature for each input spent by the Sender
func (s *Sender) SignTokenActions(raw []byte) ([][]byte, error) {
	signatures := make([][]byte, len(s.Signers))
	var err error
	for i := 0; i < len(signatures); i++ {
		signatures[i], err = s.Signers[i].Sign(raw)
		if err != nil {
			return nil, errors.Wrap(err, "failed to sign token requests")
		}
	}
	return signatures, nil
}

func getTokenData(tokens []*token.Token) []*math.G1 {
	tokenData := make([]*math.G1, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tokenData[i] = tokens[i].Data
	}
	return tokenData
}
