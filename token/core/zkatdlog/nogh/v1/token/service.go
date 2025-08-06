/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	errors2 "errors"
	"slices"

	math2 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/comm"
	utils2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var Precisions = map[token.Format]uint64{
	utils.MustGet(v1.SupportedTokenFormat(16)): 16,
	utils.MustGet(v1.SupportedTokenFormat(32)): 32,
	utils.MustGet(v1.SupportedTokenFormat(64)): 64,
}

type TokensService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*setup.PublicParams]
	IdentityDeserializer    driver.Deserializer

	OutputTokenFormat        token.Format
	SupportedTokenFormatList []token.Format
}

func NewTokensService(logger logging.Logger, publicParametersManager common.PublicParametersManager[*setup.PublicParams], identityDeserializer driver.Deserializer) (*TokensService, error) {
	// compute supported tokens
	pp := publicParametersManager.PublicParams()
	maxPrecision := pp.RangeProofParams.BitLength

	// dlog without graph hiding
	outputTokenFormat, err := supportedTokenFormat(pp, maxPrecision)
	if err != nil {
		return nil, errors.Wrapf(err, "failed computing comm token types")
	}

	supportedTokenFormatList := make([]token.Format, 0, 3*len(pp.IdemixIssuerPublicKeys))
	for _, precision := range setup.SupportedPrecisions {
		// these Precisions are supported directly
		if precision <= maxPrecision {
			format, err := supportedTokenFormat(pp, precision)
			if err != nil {
				return nil, errors.Wrapf(err, "failed computing comm token types")
			}
			supportedTokenFormatList = append(supportedTokenFormatList, format)
		}
	}

	// in addition, we support all fabtoken with precision less than maxPrecision
	for _, precision := range []uint64{16, 32, 64} {
		format, err := v1.SupportedTokenFormat(precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed computing fabtoken token format with precision [%d]", precision)
		}
		if precision <= maxPrecision {
			supportedTokenFormatList = append(supportedTokenFormatList, format)
		}
	}

	return &TokensService{
		Logger:                   logger,
		PublicParametersManager:  publicParametersManager,
		IdentityDeserializer:     identityDeserializer,
		OutputTokenFormat:        outputTokenFormat,
		SupportedTokenFormatList: supportedTokenFormatList,
	}, nil
}

func (s *TokensService) Recipients(output driver.TokenOutput) ([]driver.Identity, error) {
	tok := &Token{}
	if err := tok.Deserialize(output); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize token")
	}
	recipients, err := s.IdentityDeserializer.Recipients(tok.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recipients")
	}
	return recipients, nil
}

// Deobfuscate unmarshals a token and token metadata from raw bytes.
// We assume here that the format of the output is the default output format supported
// It checks if the un-marshalled token matches the token info. If not, it returns
// an error. Else it returns the token in cleartext and the identity of its issuer
func (s *TokensService) Deobfuscate(ctx context.Context, output driver.TokenOutput, outputMetadata driver.TokenOutputMetadata) (*token.Token, driver.Identity, []driver.Identity, token.Format, error) {
	// we support fabtoken.Type and comm.Type

	// try first comm type
	tok, issuer, recipients, format, err := s.deobfuscateAsCommType(ctx, output, outputMetadata)
	if err == nil {
		return tok, issuer, recipients, format, nil
	}
	// try fabtoken type
	return s.deobfuscateAsFabtokenType(output, outputMetadata)
}

func (s *TokensService) deobfuscateAsCommType(ctx context.Context, output driver.TokenOutput, outputMetadata driver.TokenOutputMetadata) (*token.Token, driver.Identity, []driver.Identity, token.Format, error) {
	_, metadata, tok, err := s.deserializeCommToken(ctx, output, outputMetadata, false)
	if err != nil {
		return nil, nil, nil, "", errors.Wrapf(err, "failed to deobfuscate token")
	}
	recipients, err := s.IdentityDeserializer.Recipients(tok.Owner)
	if err != nil {
		return nil, nil, nil, "", errors.Wrapf(err, "failed to get recipients")
	}
	return tok, metadata.Issuer, recipients, s.OutputTokenFormat, nil
}

func (s *TokensService) deobfuscateAsFabtokenType(output driver.TokenOutput, outputMetadata driver.TokenOutputMetadata) (*token.Token, driver.Identity, []driver.Identity, token.Format, error) {
	// TODO: refer only to the protos
	tok := &actions.Output{}
	if err := tok.Deserialize(output); err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "failed unmarshalling token")
	}

	metadata := &actions.OutputMetadata{}
	if err := metadata.Deserialize(outputMetadata); err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "failed unmarshalling token information")
	}

	recipients, err := s.IdentityDeserializer.Recipients(tok.Owner)
	if err != nil {
		return nil, nil, nil, "", errors.Wrapf(err, "failed to get recipients")
	}

	return &token.Token{
		Owner:    tok.Owner,
		Type:     tok.Type,
		Quantity: tok.Quantity,
	}, metadata.Issuer, recipients, s.OutputTokenFormat, nil
}

func (s *TokensService) SupportedTokenFormats() []token.Format {
	return s.SupportedTokenFormatList
}

func (s *TokensService) DeserializeToken(ctx context.Context, outputFormat token.Format, outputRaw []byte, metadataRaw []byte) (*Token, *Metadata, *UpgradeWitness, error) {
	// Here we have to check if what we get in input is already as expected.
	// If not, we need to check if a token upgrade is possible.
	// If not, a failure is to be returned
	if !slices.Contains(s.SupportedTokenFormatList, outputFormat) {
		return nil, nil, nil, errors.Errorf("invalid token type [%s], expected [%s]", outputFormat, s.OutputTokenFormat)
	}

	if outputFormat == s.OutputTokenFormat {
		// deserialize token with output token format
		tok, meta, err := s.deserializeTokenWithOutputTokenFormat(ctx, outputRaw, metadataRaw)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to deserialize token with output token format")
		}
		return tok, meta, nil, nil
	}

	// if we reach this point, we need to upgrade the token locally
	precision, ok := Precisions[outputFormat]
	if !ok {
		return nil, nil, nil, errors.Errorf("unsupported token format [%s]", outputFormat)
	}
	fabToken, value, err := ParseFabtokenToken(outputRaw, precision, s.PublicParametersManager.PublicParams().RangeProofParams.BitLength)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to unmarshal fabtoken token")
	}
	pp := s.PublicParametersManager.PublicParams()
	curve := math2.Curves[pp.Curve]
	tokens, meta, err := GetTokensWithWitness([]uint64{value}, fabToken.Type, pp.PedersenGenerators, curve)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to compute commitment")
	}
	return &Token{
			Owner: fabToken.Owner,
			Data:  tokens[0],
		}, &Metadata{
			Type:           fabToken.Type,
			Value:          curve.NewZrFromUint64(value),
			BlindingFactor: meta[0].BlindingFactor,
		}, &UpgradeWitness{
			FabToken:       fabToken,
			BlindingFactor: meta[0].BlindingFactor,
		}, nil
}

func (s *TokensService) deserializeTokenWithOutputTokenFormat(ctx context.Context, outputRaw []byte, metadataRaw []byte) (*Token, *Metadata, error) {
	// get zkatdlog token
	output, err := s.getOutput(ctx, outputRaw, false)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting token output")
	}

	// get metadata
	metadata := &Metadata{}
	err = metadata.Deserialize(metadataRaw)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize token metadata")
	}

	return output, metadata, nil
}

func (s *TokensService) deserializeCommToken(ctx context.Context, outputRaw []byte, metadataRaw []byte, checkOwner bool) (*Token, *Metadata, *token.Token, error) {
	// get zkatdlog token
	output, err := s.getOutput(ctx, outputRaw, checkOwner)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed getting token output")
	}

	// get metadata
	metadata := &Metadata{}
	err = metadata.Deserialize(metadataRaw)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to deserialize token metadata [%d][%v]", len(metadataRaw), metadataRaw)
	}
	pp := s.PublicParametersManager.PublicParams()

	tok, err := output.ToClear(metadata, pp)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to deserialize token")
	}
	return output, metadata, tok, nil
}

func (s *TokensService) getOutput(ctx context.Context, outputRaw []byte, checkOwner bool) (*Token, error) {
	output := &Token{}
	if err := output.Deserialize(outputRaw); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize token")
	}
	if checkOwner && len(output.Owner) == 0 {
		return nil, errors.Errorf("token owner not found in output")
	}
	if err := math.CheckElement(output.Data, s.PublicParametersManager.PublicParams().Curve); err != nil {
		return nil, errors.Wrap(err, "data in invalid in output")
	}
	return output, nil
}

func supportedTokenFormat(pp *setup.PublicParams, precision uint64) (token.Format, error) {
	hasher := utils2.NewSHA256Hasher()
	if err := errors2.Join(
		hasher.AddInt32(comm.Type),
		hasher.AddInt(int(pp.Curve)),
		hasher.AddUInt64(precision),
		hasher.AddG1s(pp.PedersenGenerators),
	); err != nil {
		return "", errors.Wrapf(err, "failed to generator token type")
	}
	return token.Format(hasher.HexDigest()), nil
}
