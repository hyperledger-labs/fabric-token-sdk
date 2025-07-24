/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

const txIdLabel tracing.LabelName = "tx_id"

type QueryTransactionsParams = ttxdb.QueryTransactionsParams

type Pagination = driver2.Pagination

type TransactionRecord = driver.TransactionRecord

type PageTransactionsIterator = driver2.PageIterator[*TransactionRecord]

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

type CheckService interface {
	Check(ctx context.Context) ([]string, error)
}

// Service is the interface for the owner service
type Service struct {
	networkProvider NetworkProvider
	tmsID           token.TMSID
	ttxStoreService *ttxdb.StoreService
	tokensService   *tokens.Service
	tmsProvider     TMSProvider
	finalityTracer  trace.Tracer
	checkService    CheckService
}

// Append adds the passed transaction to the database
func (a *Service) Append(ctx context.Context, tx *Transaction) error {
	// append request to the db
	if err := a.ttxStoreService.AppendTransactionRecord(ctx, tx.Request()); err != nil {
		return errors.WithMessagef(err, "failed appending request %s", tx.ID())
	}

	// listen to events
	net, err := a.networkProvider.GetNetwork(tx.Network(), tx.Channel())
	if err != nil {
		return errors.WithMessagef(err, "failed getting network instance for [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.DebugfContext(ctx, "register tx status listener for tx [%s:%s] at network", tx.ID(), tx.Network())

	if err := net.AddFinalityListener(tx.Namespace(), tx.ID(), common.NewFinalityListener(logger, a.tmsProvider, a.tmsID, a.ttxStoreService, a.tokensService, a.finalityTracer)); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.DebugfContext(ctx, "append done for request %s", tx.ID())
	return nil
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *Service) SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) error {
	return a.ttxStoreService.SetStatus(ctx, txID, status, message)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *Service) GetStatus(ctx context.Context, txID string) (TxStatus, string, error) {
	st, sm, err := a.ttxStoreService.GetStatus(ctx, txID)
	if err != nil {
		return Unknown, "", err
	}
	return st, sm, nil
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (a *Service) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	return a.ttxStoreService.GetTokenRequest(ctx, txID)
}

func (a *Service) AppendTransactionEndorseAck(ctx context.Context, txID string, id view.Identity, sigma []byte) error {
	return a.ttxStoreService.AddTransactionEndorsementAck(ctx, txID, id, sigma)
}

func (a *Service) GetTransactionEndorsementAcks(ctx context.Context, id string) (map[string][]byte, error) {
	return a.ttxStoreService.GetTransactionEndorsementAcks(ctx, id)
}

func (a *Service) Check(ctx context.Context) ([]string, error) {
	return a.checkService.Check(ctx)
}
