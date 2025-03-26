/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	errors2 "errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokensService struct {
	IdentityDeserializer driver.Deserializer
	OutputTokenFormat    token2.Format
}

func NewTokensService(pp *setup.PublicParams, identityDeserializer driver.Deserializer) (*TokensService, error) {
	supportedTokens, err := SupportedTokenFormat(pp.QuantityPrecision)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting supported token types")
	}
	return &TokensService{
		IdentityDeserializer: identityDeserializer,
		OutputTokenFormat:    supportedTokens,
	}, nil
}

func (s *TokensService) Recipients(output driver.TokenOutput) ([]driver.Identity, error) {
	tok := &actions.Output{}
	if err := tok.Deserialize(output); err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling token")
	}
	recipients, err := s.IdentityDeserializer.Recipients(tok.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recipients")
	}
	return recipients, nil
}

// Deobfuscate returns a deserialized token and the identity of its issuer
func (s *TokensService) Deobfuscate(output driver.TokenOutput, outputMetadata driver.TokenOutputMetadata) (*token2.Token, driver.Identity, []driver.Identity, token2.Format, error) {
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

	return &token2.Token{
		Owner:    tok.Owner,
		Type:     tok.Type,
		Quantity: tok.Quantity,
	}, metadata.Issuer, recipients, s.OutputTokenFormat, nil
}

func (s *TokensService) SupportedTokenFormats() []token2.Format {
	return []token2.Format{s.OutputTokenFormat}
}

type TokensUpgradeService struct{}

func (s *TokensUpgradeService) NewUpgradeChallenge() (driver.TokensUpgradeChallenge, error) {
	return nil, errors.New("not supported")
}

func (s *TokensUpgradeService) GenUpgradeProof(ch driver.TokensUpgradeChallenge, tokens []token2.LedgerToken, witness driver.TokensUpgradeWitness) (driver.TokensUpgradeProof, error) {
	return nil, errors.New("not supported")
}

func (s *TokensUpgradeService) CheckUpgradeProof(ch driver.TokensUpgradeChallenge, proof driver.TokensUpgradeProof, tokens []token2.LedgerToken) (bool, error) {
	return false, errors.New("not supported")
}

func SupportedTokenFormat(precision uint64) (token2.Format, error) {
	hasher := utils.NewSHA256Hasher()
	if err := errors2.Join(
		hasher.AddInt32(fabtoken.Type),
		hasher.AddString(x509.IdentityType),
		hasher.AddUInt64(precision),
	); err != nil {
		return "", errors.Wrapf(err, "failed to generator token type")
	}
	return token2.Format(hasher.HexDigest()), nil
}
