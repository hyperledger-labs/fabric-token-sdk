/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokensService struct {
	*common.TokensService
}

func NewTokensService() *TokensService {
	return &TokensService{TokensService: common.NewTokensService()}
}

// Deobfuscate returns a deserialized token and the identity of its issuer
func (s *TokensService) Deobfuscate(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, driver.Identity, error) {
	tok := &Output{}
	if err := tok.Deserialize(outputRaw); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token")
	}

	metadata := &OutputMetadata{}
	if err := metadata.Deserialize(tokenInfoRaw); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token information")
	}

	return tok.Output, metadata.Issuer, nil
}

func (s *TokensService) IsSpendable(output []byte, outputMetadata []byte) error {
	return nil
}
