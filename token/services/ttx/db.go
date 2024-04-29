/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

type QueryTransactionsParams = ttxdb.QueryTransactionsParams

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

	if err := net.AddFinalityListener(tx.Namespace(), tx.ID(), NewFinalityListener(net, a.tmsProvider, a.tmsID, a.ttxDB, a.tokenDB)); err != nil {
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

func (a *DB) AppendTransactionEndorseAck(txID string, id view.Identity, sigma []byte) error {
	return a.ttxDB.AddTransactionEndorsementAck(txID, id, sigma)
}

func (a *DB) GetTransactionEndorsementAcks(id string) (map[string][]byte, error) {
	return a.ttxDB.GetTransactionEndorsementAcks(id)
}

type FinalityListener struct {
	net         *network.Network
	tmsProvider TMSProvider
	tmsID       token.TMSID
	ttxDB       *ttxdb.DB
	tokens      *tokens.Tokens
}

func NewFinalityListener(net *network.Network, tmsProvider TMSProvider, tmsID token.TMSID, ttxDB *ttxdb.DB, tokens *tokens.Tokens) *FinalityListener {
	return &FinalityListener{
		net:         net,
		tmsProvider: tmsProvider,
		tmsID:       tmsID,
		ttxDB:       ttxDB,
		tokens:      tokens,
	}
}

func (t *FinalityListener) OnStatus(txID string, status int, message string, tokenRequestHash []byte) {
	defer func() {
		if e := recover(); e != nil {
			logger.Debugf("failed finality update for tx [%s]: [%s]", txID, e)
			if err := t.net.AddFinalityListener(t.tmsID.Namespace, txID, t); err != nil {
				panic(err)
			}
			logger.Debugf("added finality listener for tx [%s]...done", txID)
		}
	}()
	logger.Debugf("tx status changed for tx [%s]: [%s]", txID, status)
	var txStatus ttxdb.TxStatus
	switch status {
	case network.Valid:
		txStatus = ttxdb.Confirmed
		logger.Debugf("get token request for [%s]", txID)
		tokenRequestRaw, err := t.ttxDB.GetTokenRequest(txID)
		if err != nil {
			logger.Errorf("failed retrieving token request [%s]: [%s]", txID, err)
			panic(fmt.Errorf("failed retrieving token request [%s]: [%s]", txID, err))
		}
		tms, err := t.tmsProvider.GetManagementService(token.WithTMSID(t.tmsID))
		if err != nil {
			panic(fmt.Errorf("failed retrieving token request [%s]: [%s]", txID, err))
		}
		tr, err := tms.NewFullRequestFromBytes(tokenRequestRaw)
		if err != nil {
			panic(fmt.Errorf("failed retrieving token request [%s]: [%s]", txID, err))
		}
		if err := t.checkTokenRequest(txID, tr, tokenRequestHash); err != nil {
			logger.Errorf("tx [%d], %s", txID, err)
			txStatus = ttxdb.Deleted
			message = err.Error()
		}
		if txStatus != ttxdb.Deleted {
			logger.Debugf("append token request for [%s]", txID)
			if err := t.tokens.AppendRaw(t.tmsID, txID, tokenRequestRaw); err != nil {
				// at this stage though, we don't fail here because the commit pipeline is processing the tokens still
				logger.Errorf("failed to append token request to token db [%s]: [%s]", txID, err)
				panic(fmt.Errorf("failed to append token request to token db [%s]: [%s]", txID, err))
			}
			logger.Debugf("append token request for [%s], done", txID)
		}
	case network.Invalid:
		txStatus = ttxdb.Deleted
	}
	if err := t.ttxDB.SetStatus(txID, txStatus, message); err != nil {
		logger.Errorf("<message> [%s]: [%s]", txID, err)
		panic(fmt.Errorf("<message> [%s]: [%s]", txID, err))
	}
	logger.Debugf("tx status changed for tx [%s]: [%s] done", txID, status)
}

func (t *FinalityListener) checkTokenRequest(txID string, request *token.Request, reference []byte) error {
	trToSign, err := request.MarshalToSign()
	if err != nil {
		return errors.Errorf("can't get request hash '%s'", txID)
	}
	if base64.StdEncoding.EncodeToString(reference) != hash.Hashable(trToSign).String() {
		logger.Errorf("tx [%s], tr hashes [%s][%s]", txID, base64.StdEncoding.EncodeToString(reference), hash.Hashable(trToSign))
		// no further processing of the tokens of these transactions
		return errors.Errorf(
			"token requests do not match, tr hashes [%s][%s]",
			base64.StdEncoding.EncodeToString(reference),
			hash.Hashable(trToSign),
		)
	}
	return nil
}
