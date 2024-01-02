/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package owner

import (
	"fmt"
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
	TMSID token.TMSID
}

// Manager handles the databases
type Manager struct {
	sp     view.ServiceProvider
	kvs    KVS
	mutex  sync.Mutex
	owners map[string]*Owner
}

// NewManager creates a new Auditor manager.
func NewManager(sp view.ServiceProvider, kvs KVS) *Manager {
	return &Manager{
		sp:     sp,
		kvs:    kvs,
		owners: map[string]*Owner{},
	}
}

// Owner returns the Owner for the given TMS
func (cm *Manager) Owner(tms *token.ManagementService) (*Owner, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := tms.ID().String()
	logger.Debugf("get ttxdb for [%s]", id)
	c, ok := cm.owners[id]
	if !ok {
		// add an entry
		if err := cm.kvs.Put(kvs.CreateCompositeKeyOrPanic("owner", []string{id}), &Entry{
			TMSID: tms.ID(),
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to store db entry in KVS [%s]", tms.ID())
		}
		var err error
		c, err = cm.newOwner(tms)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate owner for wallet [%s]", tms.ID())
		}
		cm.owners[id] = c
	}
	return c, nil
}

func (cm *Manager) Restore() error {
	logger.Infof("restore owner dbs...")
	entries, err := cm.list()
	if err != nil {
		return errors.WithMessagef(err, "failed to list existing owners")
	}
	for _, entry := range entries {
		logger.Infof("restore owner dbs for entry [%s]...", entry.TMSID.String())
		tms := token.GetManagementService(cm.sp, token.WithTMSID(entry.TMSID))
		if tms == nil {
			return errors.Errorf("cannot find TMS [%s]", entry.TMSID)
		}
		if err := cm.restore(tms); err != nil {
			return errors.Errorf("cannot bootstrap auditdb for [%s]", entry.TMSID)
		}
		logger.Infof("restore owner dbs for entry [%s]...done", entry.TMSID.String())
	}
	logger.Infof("restore owner dbs...done")
	return nil
}

func (cm *Manager) newOwner(tms *token.ManagementService) (*Owner, error) {
	owner := &Owner{sp: cm.sp, db: ttxdb.Get(cm.sp, &tmsWallet{tms: tms})}
	net := network.GetInstance(cm.sp, tms.ID().Network, tms.ID().Channel)
	if net == nil {
		return nil, errors.Errorf("failed to get network instance for [%s:%s]", tms.ID().Network, tms.ID().Channel)
	}
	return owner, nil
}

func (cm *Manager) list() ([]Entry, error) {
	it, err := kvs.GetService(cm.sp).GetByPartialCompositeID("owner", nil)
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

func (cm *Manager) restore(tms *token.ManagementService) error {
	net := network.GetInstance(cm.sp, tms.ID().Network, tms.ID().Channel)
	if net == nil {
		return errors.Errorf("failed to get network instance for [%s:%s]", tms.ID().Network, tms.ID().Channel)
	}

	owner := New(cm.sp, tms)
	qe := owner.NewQueryExecutor()
	defer qe.Done()

	it, err := qe.Transactions(ttxdb.QueryTransactionsParams{})
	if err != nil {
		return errors.Errorf("failed to get tx iterator for [%s:%s:%s]", tms.ID().Network, tms.ID().Channel, tms.ID())
	}
	defer it.Close()

	v, err := net.Vault(tms.ID().Channel)
	if err != nil {
		return errors.Errorf("failed to get vault for [%s:%s:%s]", tms.ID().Network, tms.ID().Channel, tms.ID())
	}
	var pendingTXs []string
	type ToBeUpdated struct {
		TxID   string
		Status ttxdb.TxStatus
	}
	var toBeUpdated []ToBeUpdated
	for {
		tr, err := it.Next()
		if err != nil {
			return errors.Errorf("failed to get next tx record for [%s:%s:%s]", tms.ID().Network, tms.ID().Channel, tms.ID())
		}
		if tr == nil {
			break
		}
		if tr.Status == ttxdb.Pending {
			logger.Debugf("found pending transaction [%s] at [%s:%s]", tr.TxID, tms.ID().Network, tms.ID().Channel)
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
		if err := owner.db.SetStatus(updated.TxID, updated.Status); err != nil {
			return errors.WithMessagef(err, "failed setting status for request %s", updated.TxID)
		}
		logger.Infof("found transaction [%s] in vault with status [%d], corresponding pending transaction updated", updated.TxID, updated.Status)
	}

	logger.Infof("ownerdb [%s:%s], found [%d] pending transactions", tms.ID().Network, tms.ID().Channel, len(pendingTXs))

	for _, txID := range pendingTXs {
		if err := net.SubscribeTxStatusChanges(txID, &TxStatusChangesListener{net, owner.db}); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s:%s] for [%s]", tms.ID().Network, tms.ID().Channel, txID)
		}
	}

	return nil
}

var (
	managerType = reflect.TypeOf((*Manager)(nil))
)

// Get returns the Owner instance for the passed TMS
func Get(sp view.ServiceProvider, tms *token.ManagementService) *Owner {
	if tms == nil {
		logger.Debugf("no TMS provided")
		return nil
	}
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.Errorf("failed to get manager service: [%s]", err)
		return nil
	}
	auditor, err := s.(*Manager).Owner(tms)
	if err != nil {
		logger.Errorf("failed to get db for TMS [%s]: [%s]", tms.ID(), err)
		return nil
	}
	return auditor
}

// New returns the Owner instance for the passed TMS
func New(sp view.ServiceProvider, tms *token.ManagementService) *Owner {
	return Get(sp, tms)
}

type tmsWallet struct {
	tms *token.ManagementService
}

func (t *tmsWallet) ID() string {
	id := t.tms.ID()
	return fmt.Sprintf("%s-%s-%s", id.Network, id.Channel, id.Namespace)
}

func (t *tmsWallet) TMS() *token.ManagementService {
	return t.tms
}
