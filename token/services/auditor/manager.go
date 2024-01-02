/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

type KVS interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
}

type Entry struct {
	TMSID    token.TMSID
	WalletID string
}

// Manager handles the databases
type Manager struct {
	sp       view.ServiceProvider
	kvs      KVS
	mutex    sync.Mutex
	auditors map[string]*Auditor
}

// NewManager creates a new Auditor manager.
func NewManager(sp view.ServiceProvider, kvs KVS) *Manager {
	return &Manager{
		sp:       sp,
		kvs:      kvs,
		auditors: map[string]*Auditor{},
	}
}

// Auditor returns the Auditor for the given wallet
func (cm *Manager) Auditor(w *token.AuditorWallet) (*Auditor, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := w.TMS().ID().String() + w.ID()
	logger.Debugf("get ttxdb for [%s]", id)
	c, ok := cm.auditors[id]
	if !ok {
		// add an entry
		if err := cm.kvs.Put(kvs.CreateCompositeKeyOrPanic("auditor", []string{id}), &Entry{
			TMSID:    w.TMS().ID(),
			WalletID: w.ID(),
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to store db entry in KVS [%s:%s]", w.TMS().ID(), w.ID())
		}
		var err error
		c, err = cm.newAuditor(w)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate auditor for wallet [%s:%s]", w.TMS().ID(), w.ID())
		}
		cm.auditors[id] = c
	}
	return c, nil
}

func (cm *Manager) Restore() error {
	logger.Infof("restore audit dbs...")
	entries, err := cm.list()
	if err != nil {
		return errors.WithMessagef(err, "failed to list existing auditors")
	}
	for _, entry := range entries {
		logger.Infof("restore audit dbs for entry [%s:%s]...", entry.TMSID.String(), entry.WalletID)
		tms := token.GetManagementService(cm.sp, token.WithTMSID(entry.TMSID))
		if tms == nil {
			return errors.Errorf("cannot find TMS [%s]", entry.TMSID)
		}
		w := tms.WalletManager().AuditorWallet(entry.WalletID)
		if w == nil {
			return errors.Errorf("cannot find auditor wallet for [%s:%s]", entry.TMSID, entry.WalletID)
		}
		if err := cm.restore(w); err != nil {
			return errors.Errorf("cannot bootstrap auditdb for [%s:%s]", entry.TMSID, entry.WalletID)
		}
		logger.Infof("restore audit dbs for entry [%s:%s]...done", entry.TMSID.String(), entry.WalletID)
	}
	logger.Infof("restore audit dbs...done")
	return nil
}

func (cm *Manager) newAuditor(w *token.AuditorWallet) (*Auditor, error) {
	auditor := &Auditor{sp: cm.sp, db: ttxdb.Get(cm.sp, w)}
	net := network.GetInstance(cm.sp, w.TMS().Network(), w.TMS().Channel())
	if net == nil {
		return nil, errors.Errorf("failed to get network instance for [%s:%s]", w.TMS().Network(), w.TMS().Channel())
	}
	logger.Debugf("auditdb: register tx status listener for all tx at network [%s:%s]", w.TMS().Network(), w.TMS().Channel())
	return auditor, nil
}

func (cm *Manager) list() ([]Entry, error) {
	it, err := kvs.GetService(cm.sp).GetByPartialCompositeID("auditor", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db list iterator")
	}
	var res []Entry
	for {
		if !it.HasNext() {
			break
		}
		e := Entry{}
		if _, err := it.Next(&e); err != nil {
			return nil, errors.Wrapf(err, "failed to get db entry")
		}
		res = append(res, e)
	}
	return res, nil
}

func (cm *Manager) restore(w *token.AuditorWallet) error {
	net := network.GetInstance(cm.sp, w.TMS().Network(), w.TMS().Channel())
	if net == nil {
		return errors.Errorf("failed to get network instance for [%s:%s]", w.TMS().Network(), w.TMS().Channel())
	}

	auditor := New(cm.sp, w)
	qe := auditor.NewQueryExecutor()
	defer qe.Done()

	it, err := qe.Transactions(ttxdb.QueryTransactionsParams{})
	if err != nil {
		return errors.Errorf("failed to get tx iterator for [%s:%s:%s]", w.TMS().Network(), w.TMS().Channel(), w.ID())
	}
	defer it.Close()
	v, err := net.Vault(w.TMS().Channel())
	if err != nil {
		return errors.Errorf("failed to get vault for [%s:%s:%s]", w.TMS().Network(), w.TMS().Channel(), w.ID())
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
			return errors.Errorf("failed to get next tx record for [%s:%s:%s]", w.TMS().Network(), w.TMS().Channel(), w.ID())
		}
		if tr == nil {
			break
		}
		if tr.Status == ttxdb.Pending {
			logger.Infof("found pending transaction [%s] at [%s:%s]", tr.TxID, w.TMS().Network(), w.TMS().Channel())
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
		logger.Infof("found transaction [%s] in vault with status [%d], corresponding pending transaction updated", updated.TxID, updated.Status)
	}

	logger.Infof("auditdb [%s:%s], found [%d] pending transactions", w.TMS().Network(), w.TMS().Channel(), len(pendingTXs))

	for _, txID := range pendingTXs {
		if err := net.SubscribeTxStatusChanges(txID, &TxStatusChangesListener{net, auditor.db}); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s:%s] for [%s]", w.TMS().Network(), w.TMS().Channel(), txID)
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
