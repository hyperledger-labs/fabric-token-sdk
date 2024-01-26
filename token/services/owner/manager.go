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
	DB(w ttxdb.Wallet) (*ttxdb.DB, error)
}

// Manager handles the databases
type Manager struct {
	tmsProvider     TokenManagementServiceProvider
	networkProvider NetworkProvider
	ttxdbProvider   TTXDBProvider

	storage storage.DBEntriesStorage
	mutex   sync.Mutex
	owners  map[string]*Owner
}

// NewManager creates a new Owner manager.
func NewManager(tmsProvide TokenManagementServiceProvider, np NetworkProvider, ttxdbManager TTXDBProvider, storage storage.DBEntriesStorage) *Manager {
	return &Manager{
		tmsProvider:     tmsProvide,
		networkProvider: np,
		storage:         storage,
		ttxdbProvider:   ttxdbManager,
		owners:          map[string]*Owner{},
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
		if err := cm.storage.Put(tms.ID(), ""); err != nil {
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
	it, err := cm.storage.Iterator()
	if err != nil {
		return errors.WithMessagef(err, "failed to list existing owners")
	}
	defer func(it storage.Iterator[*storage.DBEntry]) {
		err := it.Close()
		if err != nil {
			logger.Warnf("failed to close iterator [%s]", err)
		}
	}(it)
	for {
		if !it.HasNext() {
			logger.Infof("restore owner dbs...done")
			return nil
		}
		entry, err := it.Next()
		if err != nil {
			return errors.Wrapf(err, "failed to get next entry")
		}
		logger.Infof("restore owner dbs for entry [%s]...", entry.TMSID.String())
		tms, err := cm.tmsProvider.GetManagementService(token.WithTMSID(entry.TMSID))
		if err != nil {
			return errors.WithMessagef(err, "cannot find TMS [%s]", entry.TMSID)
		}
		if err := cm.restore(tms); err != nil {
			return errors.Errorf("cannot bootstrap auditdb for [%s]", entry.TMSID)
		}
		logger.Infof("restore owner dbs for entry [%s]...done", entry.TMSID.String())
	}
}

func (cm *Manager) newOwner(tms *token.ManagementService) (*Owner, error) {
	db, err := cm.ttxdbProvider.DB(&tmsWallet{tms: tms})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s:%s]", tms.ID().Network, tms.ID().Channel)
	}
	owner := &Owner{
		networkProvider: cm.networkProvider,
		db:              db,
	}
	_, err = cm.networkProvider.GetNetwork(tms.ID().Network, tms.ID().Channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tms.ID().Network, tms.ID().Channel)
	}
	return owner, nil
}

func (cm *Manager) restore(tms *token.ManagementService) error {
	net, err := cm.networkProvider.GetNetwork(tms.ID().Network, tms.ID().Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tms.ID().Network, tms.ID().Channel)
	}

	owner, err := cm.Owner(tms)
	if err != nil {
		return errors.WithMessagef(err, "failed to get owner for [%s:%s]", tms.ID().Network, tms.ID().Channel)
	}
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
		logger.Infof("found transaction [%s] in vault with status [%s], corresponding pending transaction updated", updated.TxID, updated.Status)
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
