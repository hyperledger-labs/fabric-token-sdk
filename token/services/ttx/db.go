/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

type QueryTransactionsParams = ttxdb.QueryTransactionsParams

// QueryExecutor defines the interface for the query executor
type QueryExecutor struct {
	*ttxdb.QueryExecutor
}

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

// DB is the interface for the owner service
type DB struct {
	networkProvider NetworkProvider
	tmsID           token.TMSID
	ttxDB           *ttxdb.DB
	tokenDB         *tokens.Tokens
	tmsProvider     TMSProvider
}

// NewQueryExecutor returns a new query executor
func (a *DB) NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{QueryExecutor: a.ttxDB.NewQueryExecutor()}
}

// Append adds the passed transaction to the database
func (a *DB) Append(tx *Transaction) error {
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
	tx.TokenService()

	if err := net.SubscribeTxStatusChanges(tx.ID(), tx.Namespace(), NewTxStatusChangesListener(net, a.tmsID, a.ttxDB, a.tokenDB)); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("append done for request %s", tx.ID())
	return nil
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *DB) SetStatus(txID string, status TxStatus, statusMessage string) error {
	return a.ttxDB.SetStatus(txID, status, statusMessage)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *DB) GetStatus(txID string) (TxStatus, string, error) {
	st, sm, err := a.ttxDB.GetStatus(txID)
	if err != nil {
		return Unknown, "", err
	}
	return st, sm, nil
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (a *DB) GetTokenRequest(txID string) ([]byte, error) {
	return a.ttxDB.GetTokenRequest(txID)
}

func (a *DB) AppendTransactionEndorseAck(txID string, id view2.Identity, sigma []byte) error {
	return a.ttxDB.AddTransactionEndorsementAck(txID, id, sigma)
}

func (a *DB) GetTransactionEndorsementAcks(id string) (map[string][]byte, error) {
	return a.ttxDB.GetTransactionEndorsementAcks(id)
}

type TxStatusChangesListener struct {
	net         *network.Network
	tmsProvider TMSProvider
	tmsID       token.TMSID
	ttxDB       *ttxdb.DB
	tokens      *tokens.Tokens
}

func NewTxStatusChangesListener(net *network.Network, tmsID token.TMSID, ttxDB *ttxdb.DB, tokens *tokens.Tokens) *TxStatusChangesListener {
	return &TxStatusChangesListener{net: net, tmsID: tmsID, ttxDB: ttxDB, tokens: tokens}
}

func (t *TxStatusChangesListener) OnStatusChange(txID string, status int, statusMessage string, reference []byte) error {
	logger.Debugf("tx status changed for tx [%s]: [%s]", txID, status)
	var txStatus ttxdb.TxStatus
	switch network.ValidationCode(status) {
	case network.Valid:
		txStatus = ttxdb.Confirmed
		tokenRequestRaw, err := t.ttxDB.GetTokenRequest(txID)
		if err != nil {
			return errors.WithMessagef(err, "failed retriving token request [%s]", txID)
		}
		if len(reference) != 0 {
			tms, err := t.tmsProvider.GetManagementService(token.WithTMSID(t.tmsID))
			if err != nil {
				return err
			}
			tr, err := tms.NewFullRequestFromBytes(tokenRequestRaw)
			if err != nil {
				return err
			}
			if err := t.checkTokenRequest(txID, tr, reference); err != nil {
				return err
			}
		}
		if err := t.tokens.AppendRaw(t.tmsID, txID, tokenRequestRaw); err != nil {
			return errors.WithMessagef(err, "failed to append token request to token db [%s]", txID)
		}
	case network.Invalid:
		txStatus = ttxdb.Deleted
	}

	if err := t.ttxDB.SetStatus(txID, txStatus, statusMessage); err != nil {
		return errors.WithMessagef(err, "failed setting status for request [%s]", txID)
	}
	logger.Debugf("tx status changed for tx [%s]: %s done", txID, status)
	go func() {
		logger.Debugf("unsubscribe for tx [%s]...", txID)
		if err := t.net.UnsubscribeTxStatusChanges(txID, t); err != nil {
			logger.Errorf("failed to unsubscribe auditor tx listener for tx-id [%s]: [%s]", txID, err)
		}
		logger.Debugf("unsubscribe for tx [%s]...done", txID)
	}()
	return nil
}

func (t *TxStatusChangesListener) checkTokenRequest(txID string, request *token.Request, reference []byte) error {
	trToSign, err := request.MarshalToSign()
	if err != nil {
		return errors.Errorf("can't get request hash '%s'", txID)
	}
	if base64.StdEncoding.EncodeToString(reference) != hash.Hashable(trToSign).String() {
		logger.Errorf("tx [%s], tr hashes [%s][%s]", txID, base64.StdEncoding.EncodeToString(reference), hash.Hashable(trToSign))
		// no further processing of the tokens of these transactions
		return errors.Wrapf(
			network.ErrDiscardTX,
			"tx [%s], token requests do not match, tr hashes [%s][%s]",
			txID,
			base64.StdEncoding.EncodeToString(reference),
			hash.Hashable(trToSign),
		)
	}
	return nil
}
