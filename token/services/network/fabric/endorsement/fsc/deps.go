/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type TokenManagementSystem interface {
}

type TokenManagementSystemProvider interface {
	Get(opts ...token.ServiceOption) (TokenManagementSystem, error)
}

type Translator interface {
	AddPublicParamsDependency() error
	CommitTokenRequest(raw []byte, storeHash bool) ([]byte, error)
	Write(ctx context.Context, action any) error
}

type TranslatorProviderFunc = func(txID string, namespace string, rws *fabric.RWSet) (Translator, error)

// EndorserService defines the behaviors of the FSC's fabric endorser service that are needed by this package
//
//go:generate counterfeiter -o mock/endorser_service.go -fake-name EndorserService . EndorserService
type EndorserService interface {
	NewTransaction(context view.Context, opts ...fabric.TransactionOption) (*endorser.Transaction, error)
	ReceiveTx(ctx view.Context) (*endorser.Transaction, error)
	CollectEndorsements(ctx view.Context, tx *endorser.Transaction, timeOut time.Duration, parties ...view.Identity) error
	Endorse(tx *endorser.Transaction, identities ...view.Identity) (any, error)
}

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
