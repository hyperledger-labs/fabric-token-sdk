/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

const txIdLabel tracing.LabelName = "tx_id"

type QueryTransactionsParams = ttxdb.QueryTransactionsParams

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

type CheckService interface {
	Check(context context.Context) ([]string, error)
}

// StoreService is the interface for the owner service
type StoreService struct {
	networkProvider NetworkProvider
	tmsID           token.TMSID
	ttxDB           *ttxdb.StoreService
	tokenDB         *tokens.Tokens
	tmsProvider     TMSProvider
	finalityTracer  trace.Tracer
	checkService    CheckService
}

// Append adds the passed transaction to the database
func (a *StoreService) Append(tx *Transaction) error {
	// append request to the db
	if err := a.ttxDB.AppendTransactionRecord(tx.Request()); err != nil {
		return errors.WithMessagef(err, "failed appending request %s", tx.ID())
	}

	// listen to events
	net, err := a.networkProvider.GetNetwork(tx.Network(), tx.Channel())
	if err != nil {
		return errors.WithMessagef(err, "failed getting network instance for [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("register tx status listener for tx [%s:%s] at network", tx.ID(), tx.Network())

	if err := net.AddFinalityListener(tx.Namespace(), tx.ID(), common.NewFinalityListener(logger, a.tmsProvider, a.tmsID, a.ttxDB, a.tokenDB, a.finalityTracer)); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("append done for request %s", tx.ID())
	return nil
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *StoreService) SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) error {
	return a.ttxDB.SetStatus(ctx, txID, status, message)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *StoreService) GetStatus(txID string) (TxStatus, string, error) {
	st, sm, err := a.ttxDB.GetStatus(txID)
	if err != nil {
		return Unknown, "", err
	}
	return st, sm, nil
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (a *StoreService) GetTokenRequest(txID string) ([]byte, error) {
	return a.ttxDB.GetTokenRequest(txID)
}

func (a *StoreService) AppendTransactionEndorseAck(txID string, id view.Identity, sigma []byte) error {
	return a.ttxDB.AddTransactionEndorsementAck(txID, id, sigma)
}

func (a *StoreService) GetTransactionEndorsementAcks(id string) (map[string][]byte, error) {
	return a.ttxDB.GetTransactionEndorsementAcks(id)
}

func (a *StoreService) Check(context context.Context) ([]string, error) {
	return a.checkService.Check(context)
}
