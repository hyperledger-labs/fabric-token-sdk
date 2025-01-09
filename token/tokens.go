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

func (t *TokensService) NewConversionChallenge() ([]byte, error) {
	return t.ts.NewConversionChallenge()
}

func (t *TokensService) GenConversionProof(id []byte, tokens []token.LedgerToken) ([]byte, error) {
	return t.ts.GenConversionProof(id, tokens)
}
