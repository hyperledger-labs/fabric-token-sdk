/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"crypto/rand"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokensService struct{}

func NewTokensService() *TokensService {
	return &TokensService{}
}

func (s *TokensService) NewUpgradeChallenge() (driver.TokenUpgradeChallenge, error) {
	// generate a 32 bytes secure random slice
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}
	return key, nil
}

func (s *TokensService) GenUpgradeProof(ch driver.TokenUpgradeChallenge, tokens []token.LedgerToken) ([]byte, error) {
	return nil, nil
}

func (s *TokensService) CheckUpgradeProof(ch driver.TokenUpgradeChallenge, proof driver.TokenUpgradeProof, tokens []token.LedgerToken) (bool, error) {
	return true, nil
}
