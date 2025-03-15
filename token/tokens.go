/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// TokensService models the token service
type TokensService struct {
	ts driver.TokensService
}

// Deobfuscate processes the passed output and metadata to derive a token.Token, its issuer (if any), and its token format
func (t *TokensService) Deobfuscate(output []byte, outputMetadata []byte) (*token.Token, Identity, []Identity, token.Format, error) {
	return t.ts.Deobfuscate(output, outputMetadata)
}

// NewUpgradeChallenge generates a new upgrade challenge
func (t *TokensService) NewUpgradeChallenge() ([]byte, error) {
	return t.ts.NewUpgradeChallenge()
}

// GenUpgradeProof generates an upgrade proof for the given challenge and tokens
func (t *TokensService) GenUpgradeProof(id []byte, tokens []token.LedgerToken) ([]byte, error) {
	return t.ts.GenUpgradeProof(id, tokens)
}

// SupportedTokenFormats returns the supported token formats
func (t *TokensService) SupportedTokenFormats() []token.Format {
	return t.ts.SupportedTokenFormats()
}
