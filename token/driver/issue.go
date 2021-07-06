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
	// In addition, a set of options can be specified to further customized the issue command
	Issue(id view.Identity, typ string, values []uint64, owners [][]byte, opts *IssueOptions) (IssueAction, [][]byte, view.Identity, error)

	VerifyIssue(tr IssueAction, tokenInfos [][]byte) error

	DeserializeIssueAction(raw []byte) (IssueAction, error)
}
