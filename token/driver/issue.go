/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

// IssueOptions models the options that can be passed to the issue command
type IssueOptions struct {
	// Attributes is a container of generic options that might be driver specific
	Attributes map[interface{}]interface{}
}

// IssueService models the token issue service
type IssueService interface {
	// Issue generates an IssuerAction whose tokens are issued by the passed identity.
	// The tokens to be issued are passed as pairs (value, owner).
	// In addition, a set of options can be specified to further customize the issue command.
	// The function returns an IssuerAction, the associated metadata, and the identity of the issuer (depending on the implementation, it can be different from
	// the one passed in input).
	// The metadata is an array with an entry for each output created by the action.
	Issue(issuerIdentity view.Identity, tokenType string, values []uint64, owners [][]byte, opts *IssueOptions) (IssueAction, [][]byte, view.Identity, error)

	// VerifyIssue checks the well-formedness of the passed IssuerAction with the respect to the passed metadata
	VerifyIssue(tr IssueAction, metadata [][]byte) error

	// DeserializeIssueAction deserializes the passed bytes into an IssuerAction
	DeserializeIssueAction(raw []byte) (IssueAction, error)
}
