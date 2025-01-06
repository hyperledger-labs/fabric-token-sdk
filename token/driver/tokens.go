/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

//go:generate counterfeiter -o mock/tss.go -fake-name TokensService . TokensService

type TokensService interface {
	// SupportedTokenTypes returns the supported token types
	SupportedTokenTypes() []token.TokenFormat

	// Deobfuscate processes the passed output and metadata to derive a token.Token, its issuer (if any), and its token type tag
	Deobfuscate(output []byte, outputMetadata []byte) (*token.Token, Identity, token.TokenFormat, error)

	// ExtractMetadata extracts from the given token request metadata the metadata to the given target
	ExtractMetadata(meta *TokenRequestMetadata, target []byte) ([]byte, error)
}
