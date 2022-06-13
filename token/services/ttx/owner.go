/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
)

type txOwner struct {
	sp    view2.ServiceProvider
	tms   *token.ManagementService
	owner *owner.Owner
}

// NewOwner returns a new owner service.
func NewOwner(sp view2.ServiceProvider, tms *token.ManagementService) *txOwner {
	return &txOwner{
		sp:    sp,
		tms:   tms,
		owner: owner.New(sp, tms),
	}
}

// NewQueryExecutor returns a new query executor.
// The query executor is used to execute queries against the token transaction DB.
// The function `Done` on the query executor must be called when it is no longer needed.
func (a *txOwner) NewQueryExecutor() *owner.QueryExecutor {
	return a.owner.NewQueryExecutor()
}

// Append adds a new transaction to the token transaction database.
func (a *txOwner) Append(tx *Transaction) error {
	return a.owner.Append(tx)
}
