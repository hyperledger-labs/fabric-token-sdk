/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type TokensService struct{}

func NewTokensService() *TokensService {
	return &TokensService{}
}

func (s *TokensService) ExtractMetadata(meta *driver.TokenRequestMetadata, target []byte) ([]byte, error) {
	tokenInfoRaw := meta.GetTokenInfo(target)
	if len(tokenInfoRaw) == 0 {
		return nil, errors.Errorf("metadata for [%s] not found", Hashable(target))
	}
	return tokenInfoRaw, nil
}
