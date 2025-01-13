/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	errors2 "errors"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	fabtoken2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/math"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/comm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var precisions = map[token.Format]uint64{
	utils.MustGet(fabtoken2.SupportedTokenFormat(16)): 16,
	utils.MustGet(fabtoken2.SupportedTokenFormat(32)): 32,
	utils.MustGet(fabtoken2.SupportedTokenFormat(64)): 64,
}

type TokensService struct {
	*common.TokensService

	PublicParametersManager common.PublicParametersManager[*crypto.PublicParams]
	IdentityDeserializer    driver.Deserializer

	OutputTokenFormat               token.Format
	SupportedTokenFormatList        []token.Format
	UpgradeSupportedTokenFormatList []token.Format
}

func NewTokensService(publicParametersManager common.PublicParametersManager[*crypto.PublicParams], identityDeserializer driver.Deserializer) (*TokensService, error) {
	// compute supported tokens
	// dlog without graph hiding
	pp := publicParametersManager.PublicParams()
	maxPrecision := pp.RangeProofParams.BitLength

	outputTokenFormat, err := supportedTokenFormat(pp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed computing comm token types")
	}

	// we support all fabtoken with precision less than maxPrecision, in addition to outputTokenFormat
	supportedTokenFormatList := []token.Format{outputTokenFormat}
	var upgradeSupportedTokenFormatList []token.Format
	for _, precision := range []uint64{16, 32, 64} {
		format, err := fabtoken2.SupportedTokenFormat(precision)
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
		TokensService:                   common.NewTokensService(),
		PublicParametersManager:         publicParametersManager,
		IdentityDeserializer:            identityDeserializer,
		OutputTokenFormat:               outputTokenFormat,
		SupportedTokenFormatList:        supportedTokenFormatList,
		UpgradeSupportedTokenFormatList: upgradeSupportedTokenFormatList,
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
	return s.SupportedTokenFormatList
}

func (s *TokensService) DeserializeToken(outputFormat token.Format, outputRaw []byte, metadataRaw []byte) (*token2.Token, *token2.Metadata, *token2.ConversionWitness, error) {
	// Here we have to check if what we get in input is already as expected.
	// If not, we need to check if a token upgrade is possible.
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

func (s *TokensService) GenUpgradeProof(ch driver.TokensUpgradeChallenge, tokens []token.LedgerToken) ([]byte, error) {
	// TODO: implement
	return nil, nil
}

func (s *TokensService) CheckUpgradeProof(ch driver.TokensUpgradeChallenge, proof driver.TokensUpgradeProof, tokens []token.LedgerToken) (bool, error) {
	// TODO: implement
	return true, nil
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
		tokenTypes[i], tokenValue[i], err = s.checkFabtokenToken(&tok, precision)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to check unspent tokens")
		}
	}
	return tokenTypes, tokenValue, nil
}

func (s *TokensService) checkFabtokenToken(tok *token.LedgerToken, precision uint64) (token.Type, uint64, error) {
	maxPrecision := s.PublicParametersManager.PublicParams().RangeProofParams.BitLength
	if precision < maxPrecision {
		return "", 0, errors.Errorf("unsupported precision [%d], max [%d]", precision, maxPrecision)
	}

	typedToken, err := fabtoken.UnmarshalTypedToken(tok.Token)
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to unmarshal typed token")
	}
	fabToken, err := fabtoken.UnmarshalToken(typedToken.Token)
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to unmarshal fabtoken")
	}
	q, err := token.NewUBigQuantity(fabToken.Quantity, precision)
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to create quantity")
	}

	return fabToken.Type, q.Uint64(), nil
}
