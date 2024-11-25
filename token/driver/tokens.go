/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

//go:generate counterfeiter -o mock/tss.go -fake-name TokensService . TokensService

type TokensService interface {
	// IsSpendable returns no error if the output and its metadata are recognized as well-formed and spendable by this driver
	IsSpendable(output []byte, outputMetadata []byte) error

	// Deobfuscate processes the passed output and metadata to derive a token.Token and its issuer (if any).
	Deobfuscate(output []byte, outputMetadata []byte) (*token.Token, Identity, error)

	// ExtractMetadata extracts from the given token request metadata the metadata to the given target
	ExtractMetadata(meta *TokenRequestMetadata, target []byte) ([]byte, error)
}
