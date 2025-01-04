/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	errors2 "errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/core/fabtoken"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokensService struct {
	*common.TokensService
	TokenTypes []token2.TokenType
}

func NewTokensService(pp *PublicParams) (*TokensService, error) {
	supportedTokens, err := SupportedTokenTypes(pp.QuantityPrecision)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting supported token types")
	}
	return &TokensService{TokensService: common.NewTokensService(), TokenTypes: supportedTokens}, nil
}

// Deobfuscate returns a deserialized token and the identity of its issuer
func (s *TokensService) Deobfuscate(output []byte, outputMetadata []byte) (*token2.Token, driver.Identity, token2.TokenType, error) {
	tok := &Output{}
	if err := tok.Deserialize(output); err != nil {
		return nil, nil, "", errors.Wrap(err, "failed unmarshalling token")
	}

	metadata := &OutputMetadata{}
	if err := metadata.Deserialize(outputMetadata); err != nil {
		return nil, nil, "", errors.Wrap(err, "failed unmarshalling token information")
	}
	return &token2.Token{
		Owner:    tok.Owner,
		Type:     tok.Type,
		Quantity: tok.Quantity,
	}, metadata.Issuer, s.TokenTypes[0], nil
}

func (s *TokensService) SupportedTokenTypes() []token2.TokenType {
	return s.TokenTypes
}

func SupportedTokenTypes(precisions ...uint64) ([]token2.TokenType, error) {
	result := make([]token2.TokenType, len(precisions))
	for i, precision := range precisions {
		hasher := common.NewSHA256Hasher()
		if err := errors2.Join(
			hasher.AddInt32(fabtoken.Type),
			hasher.AddString(msp.X509Identity),
			hasher.AddUInt64(precision),
		); err != nil {
			return nil, errors.Wrapf(err, "failed to generator token type")
		}
		result[i] = token2.TokenType(hasher.HexDigest())
	}
	return result, nil
}
