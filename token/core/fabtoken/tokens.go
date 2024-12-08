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
	pp *PublicParams
}

func NewTokensService(pp *PublicParams) *TokensService {
	return &TokensService{TokensService: common.NewTokensService(), pp: pp}
}

// Deobfuscate returns a deserialized token and the identity of its issuer
func (s *TokensService) Deobfuscate(output []byte, outputMetadata []byte) (*token2.Token, driver.Identity, string, error) {
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
	}, metadata.Issuer, "", nil
}

func (s *TokensService) SupportedTokenTypes() ([]string, error) {
	// The token type is derived by combining the following elements:
	// fabtoken.Type (token type)
	// X509Identity
	// pp's QuantityPrecision
	hasher := common.NewSHA256Hasher()
	if err := errors2.Join(
		hasher.AddInt32(fabtoken.Type),
		hasher.AddString(msp.X509Identity),
		hasher.AddUInt64(s.pp.QuantityPrecision),
	); err != nil {
		return nil, err
	}
	return []string{hasher.HexDigest()}, nil
}
