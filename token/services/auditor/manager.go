/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"reflect"
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

type TokenManagementServiceProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type TTXDBProvider interface {
	DBByIDs(tmsID token.TMSID, walletID string) (*ttxdb.DB, error)
}

// Manager handles the databases
type Manager struct {
	networkProvider NetworkProvider
	ttxdbProvider   TTXDBProvider

	storage  storage.DBEntriesStorage
	mutex    sync.Mutex
	auditors map[string]*Auditor
}

// NewManager creates a new Auditor manager.
func NewManager(networkProvider NetworkProvider, ttxdbProvider TTXDBProvider, storage storage.DBEntriesStorage) *Manager {
	return &Manager{
		networkProvider: networkProvider,
		storage:         storage,
		ttxdbProvider:   ttxdbProvider,
		auditors:        map[string]*Auditor{},
	}
}

// Auditor returns the Auditor for the given wallet
func (cm *Manager) Auditor(w *token.AuditorWallet) (*Auditor, error) {
	return cm.getAuditor(w.TMS().ID(), w.ID())
}

func (cm *Manager) RestoreTMS(tmsID token.TMSID) error {
	logger.Infof("restore audit dbs...")
	it, err := cm.storage.Iterator()
	if err != nil {
		return errors.WithMessagef(err, "failed to list existing auditors")
	}
	defer func(it storage.Iterator[*storage.DBEntry]) {
		err := it.Close()
		if err != nil {
			logger.Warnf("failed to close iterator [%s][%s]", err, debug.Stack())
		}
	}(it)
	for {
		if !it.HasNext() {
			logger.Infof("restore audit dbs...done")
			return nil
		}
		entry, err := it.Next()
		if err != nil {
			return errors.Wrapf(err, "failed to get next entry for [%s:%s]...", entry.TMSID, entry.WalletID)
		}
		if entry.TMSID.Equal(tmsID) {
			logger.Infof("restore audit dbs for entry [%s:%s]...", entry.TMSID, entry.WalletID)
			if err := cm.restore(entry.TMSID, entry.WalletID); err != nil {
				return errors.Wrapf(err, "cannot bootstrap auditdb for [%s:%s]", entry.TMSID, entry.WalletID)
			}
			logger.Infof("restore audit dbs for entry [%s:%s]...done", entry.TMSID, entry.WalletID)
		}
	}
}

func (cm *Manager) getAuditor(tmsID token.TMSID, walletID string) (*Auditor, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := tmsID.String() + walletID
	logger.Debugf("get ttxdb for [%s]", id)
	c, ok := cm.auditors[id]
	if !ok {
		// add an entry
		if err := cm.storage.Put(tmsID, walletID); err != nil {
			return nil, errors.Wrapf(err, "failed to store auditor entry [%s:%s]", tmsID, walletID)
		}
		var err error
		c, err = cm.newAuditor(tmsID, walletID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate auditor for wallet [%s:%s]", tmsID, walletID)
		}
		cm.auditors[id] = c
	}
	return c, nil
}

func (cm *Manager) newAuditor(tmsID token.TMSID, walletID string) (*Auditor, error) {
	db, err := cm.ttxdbProvider.DBByIDs(tmsID, walletID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s]", walletID)
	}
	auditor := &Auditor{np: cm.networkProvider, db: db}
	_, err = cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network instance for [%s]", tmsID)
	}
	logger.Debugf("auditdb: register tx status listener for all tx at network [%s]", tmsID)
	return auditor, nil
}

func (cm *Manager) restore(tmsID token.TMSID, walletID string) error {
	net, err := cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s]", tmsID)
	}

	auditor, err := cm.getAuditor(tmsID, walletID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get auditor for [%s]", walletID)
	}
	qe := auditor.NewQueryExecutor()
	defer qe.Done()

	it, err := qe.Transactions(ttxdb.QueryTransactionsParams{})
	if err != nil {
		return errors.Errorf("failed to get tx iterator for [%s:%s]", tmsID, walletID)
	}
	defer it.Close()
	v, err := net.Vault(tmsID.Channel)
	if err != nil {
		return errors.Errorf("failed to get vault for [%s:%s]", tmsID, walletID)
	}
	type ToBeUpdated struct {
		TxID   string
		Status ttxdb.TxStatus
	}
	var toBeUpdated []ToBeUpdated
	var pendingTXs []string
	for {
		tr, err := it.Next()
		if err != nil {
			return errors.Errorf("failed to get next tx record for [%s:%s]", tmsID, walletID)
		}
		if tr == nil {
			break
		}
		if tr.Status == ttxdb.Pending {
			logger.Infof("found pending transaction [%s] at [%s]", tr.TxID, tmsID)
			found := false
			for _, txID := range pendingTXs {
				if tr.TxID == txID {
					found = true
					break
				}
			}
			if found {
				continue
			}

			// check the status of the pending transactions in the vault
			status, err := v.Status(tr.TxID)
			if err != nil {
				pendingTXs = append(pendingTXs, tr.TxID)
				continue
			}

			var txStatus ttxdb.TxStatus
			switch status {
			case network.Valid:
				txStatus = ttxdb.Confirmed
			case network.Invalid:
				txStatus = ttxdb.Deleted
			default:
				pendingTXs = append(pendingTXs, tr.TxID)
				continue
			}
			toBeUpdated = append(toBeUpdated, ToBeUpdated{
				TxID:   tr.TxID,
				Status: txStatus,
			})
		}
	}
	it.Close()
	qe.Done()

	for _, updated := range toBeUpdated {
		if err := auditor.db.SetStatus(updated.TxID, updated.Status); err != nil {
			return errors.WithMessagef(err, "failed setting status for request %s", updated.TxID)
		}
		logger.Infof("found transaction [%s] in vault with status [%s], corresponding pending transaction updated", updated.TxID, updated.Status)
	}

	logger.Infof("auditdb [%s], found [%d] pending transactions", tmsID, len(pendingTXs))

	for _, txID := range pendingTXs {
		if err := net.SubscribeTxStatusChanges(txID, &TxStatusChangesListener{net, auditor.db}); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s] for [%s]", tmsID, txID)
		}
	}

	return nil
}

var (
	managerType = reflect.TypeOf((*Manager)(nil))
)

// Get returns the Auditor instance for the passed auditor wallet
func Get(sp view.ServiceProvider, w *token.AuditorWallet) *Auditor {
	if w == nil {
		logger.Debugf("no wallet provided")
		return nil
	}
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.Errorf("failed to get manager service: [%s]", err)
		return nil
	}
	auditor, err := s.(*Manager).Auditor(w)
	if err != nil {
		logger.Errorf("failed to get db for wallet [%s:%s]: [%s]", w.TMS().ID(), w.ID(), err)
		return nil
	}
	return auditor
}

// New returns the Auditor instance for the passed auditor wallet
func New(sp view.ServiceProvider, w *token.AuditorWallet) *Auditor {
	return Get(sp, w)
}
