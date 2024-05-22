/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

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

// DeserializeToken returns a deserialized token and the identity of its issuer
func (s *TokensService) DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, driver.Identity, error) {
	tok := &token2.Token{}
	if err := json.Unmarshal(outputRaw, tok); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token")
	}

	tokInfo := &OutputMetadata{}
	if err := tokInfo.Deserialize(tokenInfoRaw); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token information")
	}

	return tok, tokInfo.Issuer, nil
}
