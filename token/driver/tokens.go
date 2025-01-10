/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

//go:generate counterfeiter -o mock/tss.go -fake-name TokensService . TokensService

type (
	ConversionChallenge []byte
	ConversionProof     []byte
)

// TokensConversionService models the token conversion service
type TokensConversionService interface {
	// NewConversionChallenge generates a new conversion challenge
	NewConversionChallenge() (ConversionChallenge, error)
	// GenConversionProof generates a conversion proof for the given challenge and tokens
	GenConversionProof(ch ConversionChallenge, tokens []token.LedgerToken) ([]byte, error)
	// CheckConversionProof checks the conversion proof for the given challenge and tokens
	CheckConversionProof(ch ConversionChallenge, proof ConversionProof, tokens []token.LedgerToken) (bool, error)
}

// TokensService models the token service
type TokensService interface {
	TokensConversionService

	// SupportedTokenFormats returns the supported token formats
	SupportedTokenFormats() []token.Format

	// Deobfuscate processes the passed output and metadata to derive the following:
	// - a token.Token,
	// - its issuer (if any),
	// - the recipients defined by Token.Owner,
	// = and the output format
	Deobfuscate(output []byte, outputMetadata []byte) (*token.Token, Identity, []Identity, token.Format, error)

	// Recipients returns the recipients of the passed output
	Recipients(output []byte) ([]Identity, error)
}
