/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package api

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

type IssueService interface {
	Issue(id view.Identity, typ string, values []uint64, owners [][]byte) (IssueAction, [][]byte, view.Identity, error)

	VerifyIssue(tr IssueAction, tokenInfos [][]byte) error

	DeserializeIssueAction(raw []byte) (IssueAction, error)
}
