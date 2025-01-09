/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// UnspendableTokenPackage is a container of unspendable tokens that should be redeemed during in an issue action
type UnspendableTokenPackage struct {
	// UnspendableTokens is a list of unspendable tokens that should be redeemed in an issue action
	UnspendableTokens []token.UnspendableTokenInWallet
	// Witness is a witness that proves the ownership of the unspendable tokens
	Witness []byte
}

// IssueOptions models the options that can be passed to the issue command
type IssueOptions struct {
	// Attributes is a container of generic options that might be driver specific
	Attributes map[interface{}]interface{}
	// UnspendableTokenPackage is a container of unspendable tokens that should be redeemed in the same issue action
	UnspendableTokenPackage *UnspendableTokenPackage
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
