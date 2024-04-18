/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type TokensService struct{}

func NewTokensService() *TokensService {
	return &TokensService{}
}

func (s *TokensService) GetTokenInfo(meta *driver.TokenRequestMetadata, target []byte) ([]byte, error) {
	tokenInfoRaw := meta.GetTokenInfo(target)
	if len(tokenInfoRaw) == 0 {
		return nil, errors.Errorf("metadata for [%s] not found", hash.Hashable(target).String())
	}
	return tokenInfoRaw, nil
}
