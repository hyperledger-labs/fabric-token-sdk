/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

// Context is an alias for view.Context
//
//go:generate counterfeiter -o mock/ctx.go -fake-name Context . Context
type Context = view.Context

// FabricTransaction is an alias for driver.Transaction
//
//go:generate counterfeiter -o mock/fabric_transaction.go -fake-name FabricTransaction . FabricTransaction
type FabricTransaction = driver.Transaction

type IdentityProvider interface {
	Identity(string) view.Identity
}

type ViewManager interface {
	InitiateView(view view.View, ctx context.Context) (interface{}, error)
}

type ViewRegistry interface {
	RegisterResponder(responder view.View, initiatedBy interface{}) error
}

// NamespaceTxProcessor models a namespace transaction processor.
type NamespaceTxProcessor interface {
	// EnableTxProcessing signals the backend to process all the transactions for the given tms id
	EnableTxProcessing(tmsID token.TMSID) error
}
