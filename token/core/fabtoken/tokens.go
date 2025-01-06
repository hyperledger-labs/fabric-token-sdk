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
	OutputTokenFormat token2.TokenFormat
}

func NewTokensService(pp *PublicParams) (*TokensService, error) {
	supportedTokens, err := supportedTokenFormat(pp.QuantityPrecision)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting supported token types")
	}
	return &TokensService{TokensService: common.NewTokensService(), OutputTokenFormat: supportedTokens}, nil
}

// Deobfuscate returns a deserialized token and the identity of its issuer
func (s *TokensService) Deobfuscate(output []byte, outputMetadata []byte) (*token2.Token, driver.Identity, token2.TokenFormat, error) {
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
	}, metadata.Issuer, s.OutputTokenFormat, nil
}

func (s *TokensService) SupportedTokenTypes() []token2.TokenFormat {
	return []token2.TokenFormat{s.OutputTokenFormat}
}

func supportedTokenFormat(precision uint64) (token2.TokenFormat, error) {
	hasher := common.NewSHA256Hasher()
	if err := errors2.Join(
		hasher.AddInt32(fabtoken.Type),
		hasher.AddString(msp.X509Identity),
		hasher.AddUInt64(precision),
	); err != nil {
		return "", errors.Wrapf(err, "failed to generator token type")
	}
	return token2.TokenFormat(hasher.HexDigest()), nil
}
