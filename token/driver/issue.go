/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// TokenConversionRequest is a request to convert tokens
type TokenConversionRequest struct {
	// Challenge is a challenge to be solved by the prover
	Challenge TokenUpgradeChallenge
	// Tokens is a list of tokens to be converted
	Tokens []token.LedgerToken
	// Proof is a proof that the prover has solved the challenge
	Proof TokenUpgradeProof
}

// IssueOptions models the options that can be passed to the issue command
type IssueOptions struct {
	// Attributes is a container of generic options that might be driver specific
	Attributes map[interface{}]interface{}
	// TokenConversionRequest is a request to convert tokens
	TokenConversionRequest *TokenConversionRequest
	// Wallet is the wallet that should be used to issue the tokens.
	Wallet IssuerWallet
}

// IssueService models the token issue service
type IssueService interface {
	// Issue generates an IssuerAction whose tokens are issued by the passed identity.
	// The tokens to be issued are passed as pairs (value, owner).
	// In addition, a set of options can be specified to further customize the issue command.
	// The function returns an IssuerAction, the associated metadata, and the identity of the issuer (depending on the implementation, it can be different from
	// the one passed in input).
	// The metadata is an array with an entry for each output created by the action.
	Issue(ctx context.Context, issuerIdentity Identity, tokenType token.Type, values []uint64, owners [][]byte, opts *IssueOptions) (IssueAction, *IssueMetadata, error)

	// VerifyIssue checks the well-formedness of the passed IssuerAction with the respect to the passed metadata
	VerifyIssue(tr IssueAction, metadata [][]byte) error

	// DeserializeIssueAction deserializes the passed bytes into an IssuerAction
	DeserializeIssueAction(raw []byte) (IssueAction, error)
}
