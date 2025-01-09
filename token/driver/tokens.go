/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

//go:generate counterfeiter -o mock/tss.go -fake-name TokensService . TokensService

type TokensService interface {
	// SupportedTokenFormats returns the supported token formats
	SupportedTokenFormats() []token.Format

	// Deobfuscate processes the passed output and metadata to derive the following:
	// - a token.Token,
	// - its issuer (if any),
	// - the recipients defined by Token.Owner,
	// = and the output format
	Deobfuscate(output []byte, outputMetadata []byte) (*token.Token, Identity, []Identity, token.Format, error)
}
