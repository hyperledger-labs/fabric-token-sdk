/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
)

type CheckTTXDB struct {
	Auditor         bool
	AuditorWalletID string
	TMSID           token.TMSID
}

// CheckTTXDBView is a view that performs consistency checks among the transaction db (either auditor or owner),
// the vault, and the backed. It reports a list of mismatch that can be used for debug purposes.
type CheckTTXDBView struct {
	*CheckTTXDB
}

func (m *CheckTTXDBView) Call(context view.Context) (interface{}, error) {
	var errorMessages []string

	tms := token.GetManagementService(context, token.WithTMSID(m.TMSID))
	assert.NotNil(tms, "failed to get default tms")
	net := network.GetInstance(context, tms.Network(), tms.Channel())
	assert.NotNil(net, "failed to get network [%s:%s]", tms.Network(), tms.Channel())
	v, err := net.Vault(tms.Namespace())
	assert.NoError(err, "failed to get vault [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())
	l, err := net.Ledger(tms.Namespace())
	assert.NoError(err, "failed to get ledger [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())
	tip := ttx.NewTransactionInfoProvider(context, tms)

	var qe *ttxdb.QueryExecutor
	if m.Auditor {
		auditorWallet := tms.WalletManager().AuditorWallet(m.AuditorWalletID)
		assert.NotNil(auditorWallet, "cannot find auditor wallet [%s]", m.AuditorWalletID)
		db := auditor.New(context, auditorWallet)
		qe = db.NewQueryExecutor().QueryExecutor
	} else {
		db := owner.New(context, tms)
		qe = db.NewQueryExecutor().QueryExecutor
	}
	defer qe.Done()
	it, err := qe.Transactions(owner.QueryTransactionsParams{})
	assert.NoError(err, "failed to get transaction iterators")
	defer it.Close()
	for {
		transactionRecord, err := it.Next()
		assert.NoError(err, "failed to get next transaction record")
		if transactionRecord == nil {
			break
		}

		// compare the status in the vault with the status of the record
		vc, err := v.Status(transactionRecord.TxID)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("failed to get vault status transaction record [%s]: [%s]", transactionRecord.TxID, err))
			continue
		}
		switch {
		case vc == network.Unknown:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is unknown for vault but not for the db [%s]", transactionRecord.TxID, transactionRecord.Status))
		case vc == network.HasDependencies:
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] has dependencies", transactionRecord.TxID))
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

		// check envelope
		if !net.ExistEnvelope(transactionRecord.TxID) {
			errorMessages = append(errorMessages, fmt.Sprintf("no envelope found for transaction record [%s]", transactionRecord.TxID))
		}
		// check metadata
		if !net.ExistTransient(transactionRecord.TxID) {
			errorMessages = append(errorMessages, fmt.Sprintf("no metadata found for transaction record [%s]", transactionRecord.TxID))
		}

		if _, err := tip.TransactionInfo(transactionRecord.TxID); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("failed to load transaction info for transaction record [%s]: [%s]", transactionRecord.TxID, err))
		}

		// check the ledger
		lVC, err := l.Status(transactionRecord.TxID)
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
		case vc == network.Busy && lVC != network.Unknown:
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("failed to get ledger transaction status for [%s]: [%s]", transactionRecord.TxID, err))
			}
			errorMessages = append(errorMessages, fmt.Sprintf("transaction record [%s] is busy for vault but not for the ledger [%d]", transactionRecord.TxID, lVC))
		}
	}

	return errorMessages, nil
}

type CheckTTXDBViewFactory struct{}

func (p *CheckTTXDBViewFactory) NewView(in []byte) (view.View, error) {
	f := &CheckTTXDBView{CheckTTXDB: &CheckTTXDB{}}
	err := json.Unmarshal(in, f.CheckTTXDB)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
