/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
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
	defaultOwnerWallet := htlc.GetWallet(context, "", token.WithTMSID(m.TMSID))
	if defaultOwnerWallet != nil {
		htlcWallet := htlc.Wallet(context, defaultOwnerWallet)
		assert.NotNil(htlcWallet, "cannot load htlc wallet")
		assert.NoError(htlcWallet.DeleteClaimedSentTokens(context), "failed to delete claimed sent tokens")
		assert.NoError(htlcWallet.DeleteExpiredReceivedTokens(context), "failed to delete expired received tokens")
	}

	// check unspent tokens
	uit, err := v.UnspentTokensIterator()
	assert.NoError(err, "failed to get unspent tokens")
	defer uit.Close()
	var unspentTokenIDs []*token2.ID
	for {
		tok, err := uit.Next()
		assert.NoError(err, "failed to get next unspent token")
		if tok == nil {
			break
		}
		unspentTokenIDs = append(unspentTokenIDs, tok.Id)
	}
	ledgerTokenContent, err := net.QueryTokens(context, tms.Namespace(), unspentTokenIDs)
	if err != nil {
		errorMessages = append(errorMessages, fmt.Sprintf("[ow:%s] failed to query tokens: [%s]", defaultOwnerWallet.ID(), err))
	} else {
		assert.Equal(len(unspentTokenIDs), len(ledgerTokenContent))
		index := 0
		assert.NoError(v.TokenVault().QueryEngine().GetTokenOutputs(unspentTokenIDs, func(id *token2.ID, tokenRaw []byte) error {
			if !bytes.Equal(ledgerTokenContent[index], tokenRaw) {
				errorMessages = append(errorMessages, fmt.Sprintf("[ow:%s] token content does not match at [%d], [%s]!=[%s]",
					defaultOwnerWallet.ID(),
					index,
					hash.Hashable(ledgerTokenContent[index]), hash.Hashable(tokenRaw)))
			}
			index++
			return nil
		}), "failed to match ledger token content with local")
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

type PruneInvalidUnspentTokens struct {
	TMSID token.TMSID
}

type PruneInvalidUnspentTokensView struct {
	*PruneInvalidUnspentTokens
}

func (p *PruneInvalidUnspentTokensView) Call(context view.Context) (interface{}, error) {
	net := network.GetInstance(context, p.TMSID.Network, p.TMSID.Channel)
	assert.NotNil(net, "cannot find network [%s:%s]", p.TMSID.Network, p.TMSID.Channel)
	vault, err := net.Vault(p.TMSID.Namespace)
	assert.NoError(err, "failed to get vault for [%s:%s:%s]", p.TMSID.Network, p.TMSID.Channel, p.TMSID.Namespace)

	return vault.PruneInvalidUnspentTokens(context)
}

type PruneInvalidUnspentTokensViewFactory struct{}

func (p *PruneInvalidUnspentTokensViewFactory) NewView(in []byte) (view.View, error) {
	f := &PruneInvalidUnspentTokensView{PruneInvalidUnspentTokens: &PruneInvalidUnspentTokens{}}
	err := json.Unmarshal(in, f.PruneInvalidUnspentTokens)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type ListVaultUnspentTokens struct {
	TMSID token.TMSID
}

type ListVaultUnspentTokensView struct {
	*ListVaultUnspentTokens
}

func (l *ListVaultUnspentTokensView) Call(context view.Context) (interface{}, error) {
	net := network.GetInstance(context, l.TMSID.Network, l.TMSID.Channel)
	assert.NotNil(net, "cannot find network [%s:%s]", l.TMSID.Network, l.TMSID.Channel)
	vault, err := net.Vault(l.TMSID.Namespace)
	assert.NoError(err, "failed to get vault for [%s:%s:%s]", l.TMSID.Network, l.TMSID.Channel, l.TMSID.Namespace)

	return vault.ListUnspentTokens()
}

type ListVaultUnspentTokensViewFactory struct{}

func (l *ListVaultUnspentTokensViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListVaultUnspentTokensView{ListVaultUnspentTokens: &ListVaultUnspentTokens{}}
	err := json.Unmarshal(in, f.ListVaultUnspentTokens)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type CheckIfExistsInVault struct {
	TMSID token.TMSID
	IDs   []*token2.ID
}

type CheckIfExistsInVaultView struct {
	*CheckIfExistsInVault
}

func (c *CheckIfExistsInVaultView) Call(context view.Context) (interface{}, error) {
	net := network.GetInstance(context, c.TMSID.Network, c.TMSID.Channel)
	assert.NotNil(net, "cannot find network [%s:%s]", c.TMSID.Network, c.TMSID.Channel)
	vault, err := net.Vault(c.TMSID.Namespace)
	assert.NoError(err, "failed to get vault for [%s:%s:%s]", c.TMSID.Network, c.TMSID.Channel, c.TMSID.Namespace)
	qe := vault.TokenVault().QueryEngine()
	var IDs []*token2.ID
	count := 0
	assert.NoError(qe.GetTokenOutputs(c.IDs, func(id *token2.ID, tokenRaw []byte) error {
		if len(tokenRaw) == 0 {
			return errors.Errorf("token id [%s] is nil", id)
		}
		IDs = append(IDs, id)
		count++
		return nil
	}), "failed to match tokens")
	assert.Equal(len(c.IDs), count, "got a mismatch; count is [%d] while there are [%d] ids", count, len(c.IDs))
	return IDs, err
}

type CheckIfExistsInVaultViewFactory struct {
}

func (c *CheckIfExistsInVaultViewFactory) NewView(in []byte) (view.View, error) {
	f := &CheckIfExistsInVaultView{CheckIfExistsInVault: &CheckIfExistsInVault{}}
	err := json.Unmarshal(in, f.CheckIfExistsInVault)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
