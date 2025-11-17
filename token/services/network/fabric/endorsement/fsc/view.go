/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

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
