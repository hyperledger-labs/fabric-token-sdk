/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

//go:generate counterfeiter -o mock/tss.go -fake-name TokensService . TokensService

type (
	// TokensUpgradeChallenge is the challenge the issuer generates to make sure the client is not cheating
	TokensUpgradeChallenge = []byte
	// TokensUpgradeProof is the proof generated with the respect to a given challenge to prove the validity of the tokens to be upgrade
	TokensUpgradeProof = []byte

	// TokenOutput models an output on the edger
	TokenOutput []byte
	// TokenOutputMetadata models the metadata of an output on the ledger
	TokenOutputMetadata []byte
)

// TokensUpgradeService models the token update service
type TokensUpgradeService interface {
	// NewUpgradeChallenge generates a new upgrade challenge
	NewUpgradeChallenge() (TokensUpgradeChallenge, error)
	// GenUpgradeProof generates an upgrade proof for the given challenge and tokens
	GenUpgradeProof(ch TokensUpgradeChallenge, tokens []token.LedgerToken) ([]byte, error)
	// CheckUpgradeProof checks the upgrade proof for the given challenge and tokens
	CheckUpgradeProof(ch TokensUpgradeChallenge, proof TokensUpgradeProof, tokens []token.LedgerToken) (bool, error)
}

// TokensService models the token service
type TokensService interface {
	TokensUpgradeService

	// SupportedTokenFormats returns the supported token formats
	SupportedTokenFormats() []token.Format

	// Deobfuscate processes the passed output and metadata to derive the following:
	// - a token.Token,
	// - its issuer (if any),
	// - the recipients defined by Token.Owner,
	// = and the output format
	Deobfuscate(output TokenOutput, outputMetadata TokenOutputMetadata) (*token.Token, Identity, []Identity, token.Format, error)

	// Recipients returns the recipients of the passed output
	Recipients(output TokenOutput) ([]Identity, error)
}
