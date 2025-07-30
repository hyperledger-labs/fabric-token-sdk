/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transfer

import (
	"context"

	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger()

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
	if len(values) != len(owners) {
		return nil, nil, errors.Errorf("cannot generate transfer: number of values [%d] does not match number of recipients [%d]", len(values), len(owners))
	}
	logger.DebugfContext(ctx, "Get token data for %d inputs", len(s.Inputs))
	in := getTokenData(s.Inputs)
	intw := make([]*token.Metadata, len(s.InputInformation))
	for i := range len(s.InputInformation) {
		if s.InputInformation[0].Type != s.InputInformation[i].Type {
			return nil, nil, errors.New("cannot generate transfer: please choose inputs of the same token type")
		}
		intw[i] = &token.Metadata{
			Value:          s.InputInformation[i].Value,
			Type:           s.InputInformation[i].Type,
			BlindingFactor: s.InputInformation[i].BlindingFactor,
		}
	}
	logger.DebugfContext(ctx, "Get tokens with witness")
	out, outtw, err := token.GetTokensWithWitness(values, s.InputInformation[0].Type, s.PublicParams.PedersenGenerators, math.Curves[s.PublicParams.Curve])
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate transfer")
	}
	logger.DebugfContext(ctx, "Create new prover")
	prover, err := NewProver(intw, outtw, in, out, s.PublicParams)
	if err != nil {
		return nil, nil, errors.New("cannot generate transfer")
	}
	logger.DebugfContext(ctx, "Prove")
	proof, err := prover.Prove()
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate zero-knowledge proof for transfer")
	}

	logger.DebugfContext(ctx, "Create new transfer")
	transfer, err := NewTransfer(s.InputIDs, s.Inputs, out, owners, proof)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to produce transfer action")
	}
	inf := make([]*token.Metadata, len(owners))
	for i := 0; i < len(inf); i++ {
		inf[i] = &token.Metadata{
			Type:           s.InputInformation[0].Type,
			Value:          outtw[i].Value,
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
	for i := range tokens {
		tokenData[i] = tokens[i].Data
	}
	return tokenData
}
