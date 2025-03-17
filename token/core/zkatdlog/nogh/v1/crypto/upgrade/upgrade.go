/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade

import (
	"crypto/rand"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	ChallengeSize = 32
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) NewUpgradeChallenge() (driver.TokensUpgradeChallenge, error) {
	// generate a 32 bytes secure random slice
	key := make([]byte, ChallengeSize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}
	// rand.Read guarantees that len(key) == ChallengeSize, let's check it anyway
	if len(key) != ChallengeSize {
		return nil, errors.Errorf("invalid key size, got only [%d], expected [%d]", len(key), ChallengeSize)
	}
	return key, nil
}

// GenUpgradeProof does the following: For each token in input, it signs the concatenation of the challenge and the tokens to be upgraded.
// These signatures are then added to the proof
func (s *Service) GenUpgradeProof(ch driver.TokensUpgradeChallenge, tokens []token.LedgerToken) ([]byte, error) {
	// TODO: implement
	return nil, nil
}

func (s *Service) CheckUpgradeProof(ch driver.TokensUpgradeChallenge, proof driver.TokensUpgradeProof, tokens []token.LedgerToken) (bool, error) {
	// TODO: implement
	return true, nil
}
