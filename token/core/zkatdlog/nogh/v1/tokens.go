/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	errors2 "errors"
	"slices"

	math2 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/comm"
	utils2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var precisions = map[token.Format]uint64{
	utils.MustGet(v1.SupportedTokenFormat(16)): 16,
	utils.MustGet(v1.SupportedTokenFormat(32)): 32,
	utils.MustGet(v1.SupportedTokenFormat(64)): 64,
}

type TokensService struct {
	*common.TokensService

	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*crypto.PublicParams]
	IdentityDeserializer    driver.Deserializer

	OutputTokenFormat               token.Format
	SupportedTokenFormatList        []token.Format
	UpgradeSupportedTokenFormatList []token.Format
}

func NewTokensService(logger logging.Logger, publicParametersManager common.PublicParametersManager[*crypto.PublicParams], identityDeserializer driver.Deserializer) (*TokensService, error) {
	// compute supported tokens
	pp := publicParametersManager.PublicParams()
	maxPrecision := pp.RangeProofParams.BitLength

	// dlog without graph hiding
	outputTokenFormat, err := supportedTokenFormat(pp, maxPrecision)
	if err != nil {
		return nil, errors.Wrapf(err, "failed computing comm token types")
	}

	supportedTokenFormatList := make([]token.Format, 0, 3*len(pp.IdemixIssuerPublicKeys))
	for _, precision := range crypto.SupportedPrecisions {
		// these precisions are supported directly
		if precision <= maxPrecision {
			format, err := supportedTokenFormat(pp, precision)
			if err != nil {
				return nil, errors.Wrapf(err, "failed computing comm token types")
			}
			supportedTokenFormatList = append(supportedTokenFormatList, format)
		}
	}

	// in addition, we support all fabtoken with precision less than maxPrecision
	var upgradeSupportedTokenFormatList []token.Format
	for _, precision := range []uint64{16, 32, 64} {
		format, err := v1.SupportedTokenFormat(precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed computing fabtoken token format with precision [%d]", precision)
		}
		if precision <= maxPrecision {
			supportedTokenFormatList = append(supportedTokenFormatList, format)
		} else {
			upgradeSupportedTokenFormatList = append(upgradeSupportedTokenFormatList, format)
		}
	}

	return &TokensService{
		Logger:                          logger,
		TokensService:                   common.NewTokensService(),
		PublicParametersManager:         publicParametersManager,
		IdentityDeserializer:            identityDeserializer,
		OutputTokenFormat:               outputTokenFormat,
		SupportedTokenFormatList:        supportedTokenFormatList,
		UpgradeSupportedTokenFormatList: upgradeSupportedTokenFormatList,
	}, nil
}

func (s *TokensService) Recipients(output driver.TokenOutput) ([]driver.Identity, error) {
	tok := &token2.Token{}
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
func (s *TokensService) Deobfuscate(output driver.TokenOutput, outputMetadata driver.TokenOutputMetadata) (*token.Token, driver.Identity, []driver.Identity, token.Format, error) {
	// we support fabtoken.Type and comm.Type

	// try first comm type
	tok, issuer, recipients, format, err := s.deobfuscateAsCommType(output, outputMetadata)
	if err == nil {
		return tok, issuer, recipients, format, nil
	}
	// try fabtoken type
	return s.deobfuscateAsFabtokenType(output, outputMetadata)
}

func (s *TokensService) deobfuscateAsCommType(output driver.TokenOutput, outputMetadata driver.TokenOutputMetadata) (*token.Token, driver.Identity, []driver.Identity, token.Format, error) {
	_, metadata, tok, err := s.deserializeCommToken(output, outputMetadata, false)
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
	tok := &core.Output{}
	if err := tok.Deserialize(output); err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "failed unmarshalling token")
	}

	metadata := &core.OutputMetadata{}
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

func (s *TokensService) DeserializeToken(outputFormat token.Format, outputRaw []byte, metadataRaw []byte) (*token2.Token, *token2.Metadata, *token2.UpgradeWitness, error) {
	// Here we have to check if what we get in input is already as expected.
	// If not, we need to check if a token upgrade is possible.
	// If not, a failure is to be returned
	if !slices.Contains(s.SupportedTokenFormatList, outputFormat) {
		return nil, nil, nil, errors.Errorf("invalid token type [%s], expected [%s]", outputFormat, s.OutputTokenFormat)
	}

	if outputFormat == s.OutputTokenFormat {
		// deserialize token with output token format
		tok, meta, err := s.deserializeTokenWithOutputTokenFormat(outputRaw, metadataRaw)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to deserialize token with output token format")
		}
		return tok, meta, nil, nil
	}

	// if we reach this point, we need to upgrade the token locally
	precision, ok := precisions[outputFormat]
	if !ok {
		return nil, nil, nil, errors.Errorf("unsupported token format [%s]", outputFormat)
	}
	fabToken, value, err := s.checkFabtokenToken(outputRaw, precision)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to unmarshal fabtoken token")
	}
	pp := s.PublicParametersManager.PublicParams()
	curve := math2.Curves[pp.Curve]
	tokens, meta, err := token2.GetTokensWithWitness([]uint64{value}, fabToken.Type, pp.PedersenGenerators, curve)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to compute commitment")
	}
	return &token2.Token{
			Owner: fabToken.Owner,
			Data:  tokens[0],
		}, &token2.Metadata{
			Type:           fabToken.Type,
			Value:          curve.NewZrFromUint64(value),
			BlindingFactor: meta[0].BlindingFactor,
		}, &token2.UpgradeWitness{
			FabToken:       fabToken,
			BlindingFactor: meta[0].BlindingFactor,
		}, nil
}

func (s *TokensService) GenUpgradeProof(ch driver.TokensUpgradeChallenge, tokens []token.LedgerToken) ([]byte, error) {
	// TODO: implement
	return nil, nil
}

func (s *TokensService) CheckUpgradeProof(ch driver.TokensUpgradeChallenge, proof driver.TokensUpgradeProof, tokens []token.LedgerToken) (bool, error) {
	// TODO: implement
	return true, nil
}

func (s *TokensService) deserializeTokenWithOutputTokenFormat(outputRaw []byte, metadataRaw []byte) (*token2.Token, *token2.Metadata, error) {
	// get zkatdlog token
	output, err := s.getOutput(outputRaw, false)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting token output")
	}

	// get metadata
	metadata := &token2.Metadata{}
	err = metadata.Deserialize(metadataRaw)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deserialize token metadata")
	}

	return output, metadata, nil
}

func (s *TokensService) deserializeCommToken(outputRaw []byte, metadataRaw []byte, checkOwner bool) (*token2.Token, *token2.Metadata, *token.Token, error) {
	// get zkatdlog token
	output, err := s.getOutput(outputRaw, checkOwner)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed getting token output")
	}

	// get metadata
	metadata := &token2.Metadata{}
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

func (s *TokensService) getOutput(outputRaw []byte, checkOwner bool) (*token2.Token, error) {
	output := &token2.Token{}
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

func supportedTokenFormat(pp *crypto.PublicParams, precision uint64) (token.Format, error) {
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

func (s *TokensService) ProcessTokensUpgradeRequest(utp *driver.TokenUpgradeRequest) ([]token.Type, []uint64, error) {
	if utp == nil {
		return nil, nil, errors.New("nil token upgrade request")
	}

	// check that each token doesn't have a supported format
	for _, tok := range utp.Tokens {
		if !slices.Contains(s.UpgradeSupportedTokenFormatList, tok.Format) {
			return nil, nil, errors.Errorf("upgrade of unsupported token format [%s] requested", tok.Format)
		}
	}

	// check the upgrade proof
	ok, err := s.CheckUpgradeProof(utp.Challenge, utp.Proof, utp.Tokens)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to check upgrade proof")
	}
	if !ok {
		return nil, nil, errors.New("invalid upgrade proof")
	}

	// for each token, extract type and value
	tokenTypes := make([]token.Type, len(utp.Tokens))
	tokenValue := make([]uint64, len(utp.Tokens))
	for i, tok := range utp.Tokens {
		precision, ok := precisions[tok.Format]
		if !ok {
			return nil, nil, errors.Errorf("unsupported token format [%s]", tok.Format)
		}
		fabToken, v, err := s.checkFabtokenToken(tok.Token, precision)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to check unspent tokens")
		}
		tokenTypes[i] = fabToken.Type
		tokenValue[i] = v
	}
	return tokenTypes, tokenValue, nil
}

func (s *TokensService) checkFabtokenToken(tok []byte, precision uint64) (*core.Output, uint64, error) {
	maxPrecision := s.PublicParametersManager.PublicParams().RangeProofParams.BitLength
	if precision < maxPrecision {
		return nil, 0, errors.Errorf("unsupported precision [%d], max [%d]", precision, maxPrecision)
	}

	output := &core.Output{}
	err := output.Deserialize(tok)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to unmarshal fabtoken")
	}
	q, err := token.NewUBigQuantity(output.Quantity, precision)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to create quantity")
	}

	return output, q.Uint64(), nil
}
