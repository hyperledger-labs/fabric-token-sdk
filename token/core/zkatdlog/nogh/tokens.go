/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	errors2 "errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/math"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/comm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokensService struct {
	*common.TokensService
	PublicParametersManager common.PublicParametersManager[*crypto.PublicParams]
	OutputTokenFormat       token.Format
	IdentityDeserializer    driver.Deserializer
}

func NewTokensService(publicParametersManager common.PublicParametersManager[*crypto.PublicParams], identityDeserializer driver.Deserializer) (*TokensService, error) {
	// compute supported tokens
	// dlog without graph hiding
	commTokenTypes, err := supportedTokenFormat(publicParametersManager.PublicParams())
	if err != nil {
		return nil, errors.Wrapf(err, "failed computing comm token types")
	}

	return &TokensService{
		TokensService:           common.NewTokensService(),
		PublicParametersManager: publicParametersManager,
		IdentityDeserializer:    identityDeserializer,
		OutputTokenFormat:       commTokenTypes,
	}, nil
}

func (s *TokensService) Recipients(output []byte) ([]driver.Identity, error) {
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

// Deobfuscate unmarshals a token and token info from raw bytes
// It checks if the un-marshalled token matches the token info. If not, it returns
// an error. Else it returns the token in cleartext and the identity of its issuer
func (s *TokensService) Deobfuscate(output []byte, outputMetadata []byte) (*token.Token, driver.Identity, []driver.Identity, token.Format, error) {
	_, metadata, tok, err := s.deserializeToken(output, outputMetadata, false)
	if err != nil {
		return nil, nil, nil, "", errors.Wrapf(err, "failed to deobfuscate token")
	}
	recipients, err := s.IdentityDeserializer.Recipients(tok.Owner)
	if err != nil {
		return nil, nil, nil, "", errors.Wrapf(err, "failed to get recipients")
	}
	return tok, metadata.Issuer, recipients, s.OutputTokenFormat, nil
}

func (s *TokensService) SupportedTokenFormats() []token.Format {
	return []token.Format{s.OutputTokenFormat}
}

func (s *TokensService) DeserializeToken(outputFormat token.Format, outputRaw []byte, metadataRaw []byte) (*token2.Token, *token2.Metadata, *token2.ConversionWitness, error) {
	// Here we have to check if what we get in input is already as expected.
	// If not, we need to check if a conversion is possible.
	// If not, a failure is to be returned
	if outputFormat != s.OutputTokenFormat {
		return nil, nil, nil, errors.Errorf("invalid token type [%s], expected [%s]", outputFormat, s.OutputTokenFormat)
	}

	// get zkatdlog token
	output, err := s.getOutput(outputRaw, false)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed getting token output")
	}

	// get metadata
	metadata := &token2.Metadata{}
	err = metadata.Deserialize(metadataRaw)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to deserialize token metadata")
	}

	return output, metadata, nil, nil
}

func (s *TokensService) deserializeToken(outputRaw []byte, metadataRaw []byte, checkOwner bool) (*token2.Token, *token2.Metadata, *token.Token, error) {
	// get zkatdlog token
	output, err := s.getOutput(outputRaw, checkOwner)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed getting token output")
	}

	// get metadata
	metadata := &token2.Metadata{}
	err = metadata.Deserialize(metadataRaw)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to deserialize token metadata")
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

func supportedTokenFormat(pp *crypto.PublicParams) (token.Format, error) {
	hasher := common.NewSHA256Hasher()
	if err := errors2.Join(
		hasher.AddInt32(comm.Type),
		hasher.AddInt(int(pp.Curve)),
		hasher.AddG1s(pp.PedersenGenerators),
		hasher.AddInt(int(pp.IdemixCurveID)),
		hasher.AddBytes(pp.IdemixIssuerPK),
	); err != nil {
		return "", errors.Wrapf(err, "failed to generator token type")
	}
	return token.Format(hasher.HexDigest()), nil
}

func (s *TokensService) CheckUnspendableTokens(tokens []token.UnspendableTokenInWallet) {

}
