/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

const txIdLabel tracing.LabelName = "tx_id"

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
	finalityTracer  trace.Tracer
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

	if err := net.AddFinalityListener(tx.Namespace(), tx.ID(), common.NewFinalityListener(logger, a.tmsProvider, a.tmsID, a.ttxDB, a.tokenDB, a.finalityTracer)); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.Debugf("append done for request %s", tx.ID())
	return nil
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *DB) SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) error {
	return a.ttxDB.SetStatus(ctx, txID, status, message)
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

func (a *DB) Check(ctx view.Context) ([]string, error) {
	var errorMessages []string

	tms, err := a.tmsProvider.GetManagementService(token.WithTMSID(a.tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting tms [%s]", a.tmsID)
	}
	net, err := a.networkProvider.GetNetwork(tms.Network(), tms.Channel())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network [%s]", tms.ID())
	}
	v, err := net.Vault(tms.Namespace())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault [%s]", tms.ID())
	}
	tv, err := net.TokenVault(tms.Namespace())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault [%s]", tms.ID())
	}
	l, err := net.Ledger()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ledger [%s]", tms.ID())
	}

	var tokenDB TokenTransactionDB
	it, err := a.ttxDB.Transactions(driver.QueryTransactionsParams{})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed querying transactions [%s]", tms.ID())
	}
	defer it.Close()
	for {
		transactionRecord, err := it.Next()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed querying transactions [%s]", tms.ID())
		}
		if transactionRecord == nil {
			break
		}

		// compare the status in the vault with the status of the record
		vc, _, err := v.Status(transactionRecord.TxID)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("failed to get vault status transaction record [%s]: [%s]", transactionRecord.TxID, err))
			continue
		}
		switch {
		case vc == network.Unknown:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is unknown for vault but not for the db [%s]", transactionRecord.TxID, driver.TxStatusMessage[transactionRecord.Status]))
		case vc == network.Valid && transactionRecord.Status == ttxdb.Pending:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is valid for vault but pending for the db", transactionRecord.TxID))
		case vc == network.Valid && transactionRecord.Status == ttxdb.Deleted:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is valid for vault but deleted for the db", transactionRecord.TxID))
		case vc == network.Invalid && transactionRecord.Status == ttxdb.Confirmed:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is invalid for vault but confirmed for the db", transactionRecord.TxID))
		case vc == network.Invalid && transactionRecord.Status == ttxdb.Pending:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is invalid for vault but pending for the db", transactionRecord.TxID))
		case vc == network.Busy && transactionRecord.Status == ttxdb.Confirmed:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is busy for vault but confirmed for the db", transactionRecord.TxID))
		case vc == network.Busy && transactionRecord.Status == ttxdb.Deleted:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is busy for vault but deleted for the db", transactionRecord.TxID))
		}

		tokenRequest, err := tokenDB.GetTokenRequest(transactionRecord.TxID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting token request [%s]", transactionRecord.TxID)
		}
		if tokenRequest == nil {
			return nil, errors.Errorf("token request [%s] is nil", transactionRecord.TxID)
		}

		// check the ledger
		lVC, _, err := l.Status(transactionRecord.TxID)
		if err != nil {
			lVC = network.Unknown
		}
		switch {
		case vc == network.Valid && lVC != network.Valid:
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to get ledger transaction status for [%s]: [%s]", transactionRecord.TxID, err))
			}
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is valid for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
		case vc == network.Invalid && lVC != network.Invalid:
			if lVC != network.Unknown || transactionRecord.Status != ttxdb.Deleted {
				if err != nil {
					errorMessages = append(errorMessages, fmt.Sprintf("failed to get ledger transaction status for [%s]: [%s]", transactionRecord.TxID, err))
				}
				errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is invalid for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
			}
		case vc == network.Unknown && lVC != network.Unknown:
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to get ledger transaction status for [%s]: [%s]", transactionRecord.TxID, err))
			}
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is unknown for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
		case vc == network.Busy && lVC == network.Busy:
			// this is fine, let's continue
		case vc == network.Busy && lVC != network.Unknown:
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to get ledger transaction status for [%s]: [%s]", transactionRecord.TxID, err))
			}
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is busy for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
		}
	}

	// Match unspent tokens with the ledger
	// but first delete the claimed tokens
	// TODO: check all owner wallets
	// defaultOwnerWallet := tms.WalletManager().OwnerWallet("")
	// if defaultOwnerWallet != nil {
	// 	htlcWallet := htlc.Wallet(context, defaultOwnerWallet)
	// 	assert.NotNil(htlcWallet, "cannot load htlc wallet")
	// 	assert.NoError(htlcWallet.DeleteClaimedSentTokens(context), "failed to delete claimed sent tokens")
	// 	assert.NoError(htlcWallet.DeleteExpiredReceivedTokens(context), "failed to delete expired received tokens")
	// }

	// check unspent tokens
	uit, err := tv.QueryEngine().UnspentTokensIterator()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed querying utxo engine")
	}
	defer uit.Close()
	var unspentTokenIDs []*token2.ID
	for {
		tok, err := uit.Next()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed querying next unspent token")
		}
		if tok == nil {
			break
		}
		unspentTokenIDs = append(unspentTokenIDs, tok.Id)
	}
	ledgerTokenContent, err := net.QueryTokens(ctx, tms.Namespace(), unspentTokenIDs)
	if err != nil {
		errorMessages = append(errorMessages, fmt.Sprintf("failed to query tokens: [%s]", err))
	} else {
		if len(unspentTokenIDs) != len(ledgerTokenContent) {
			return nil, errors.Errorf("length diffrence")
		}
		index := 0
		if err := tv.QueryEngine().GetTokenOutputs(unspentTokenIDs, func(id *token2.ID, tokenRaw []byte) error {
			for _, content := range ledgerTokenContent {
				if bytes.Equal(content, tokenRaw) {
					return nil
				}
			}

			errorMessages = append(errorMessages, fmt.Sprintf("token content does not match at [%s][%d], [%s]", id, index, hash.Hashable(tokenRaw)))
			index++
			return nil
		}); err != nil {
			return nil, errors.WithMessagef(err, "failed to match ledger token content with local")
		}
	}
	return errorMessages, nil
}
