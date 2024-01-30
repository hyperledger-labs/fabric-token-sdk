/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package owner

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

type TTXDBProvider interface {
	DBByTMSId(id token.TMSID) (*ttxdb.DB, error)
}

// Manager handles the databases
type Manager struct {
	networkProvider NetworkProvider
	ttxdbProvider   TTXDBProvider

	storage storage.DBEntriesStorage
	mutex   sync.Mutex
	owners  map[string]*Owner
}

// NewManager creates a new Owner manager.
func NewManager(np NetworkProvider, ttxdbManager TTXDBProvider, storage storage.DBEntriesStorage) *Manager {
	return &Manager{
		networkProvider: np,
		storage:         storage,
		ttxdbProvider:   ttxdbManager,
		owners:          map[string]*Owner{},
	}
}

// Owner returns the Owner for the given TMS
func (cm *Manager) Owner(tmsID token.TMSID) (*Owner, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := tmsID.String()
	logger.Debugf("get ttxdb for [%s]", id)
	c, ok := cm.owners[id]
	if !ok {
		// add an entry
		if err := cm.storage.Put(tmsID, ""); err != nil {
			return nil, errors.Wrapf(err, "failed to store db entry in KVS [%s]", tmsID)
		}
		var err error
		c, err = cm.newOwner(tmsID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate owner for wallet [%s]", tmsID)
		}
		cm.owners[id] = c
	}
	return c, nil
}

func (cm *Manager) newOwner(tmsID token.TMSID) (*Owner, error) {
	db, err := cm.ttxdbProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s:%s]", tmsID.Network, tmsID.Channel)
	}
	owner := &Owner{
		networkProvider: cm.networkProvider,
		db:              db,
	}
	_, err = cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
	}
	return owner, nil
}

func (cm *Manager) RestoreTMS(tmsID token.TMSID) error {
	net, err := cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
	}

	owner, err := cm.Owner(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get owner for [%s:%s]", tmsID.Network, tmsID.Channel)
	}
	qe := owner.NewQueryExecutor()
	defer qe.Done()

	it, err := qe.Transactions(ttxdb.QueryTransactionsParams{})
	if err != nil {
		return errors.Errorf("failed to get tx iterator for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID)
	}
	defer it.Close()

	v, err := net.Vault(tmsID.Channel)
	if err != nil {
		return errors.Errorf("failed to get vault for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID)
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
			return errors.Errorf("failed to get next tx record for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID)
		}
		if tr == nil {
			break
		}
		if tr.Status == ttxdb.Pending {
			logger.Debugf("found pending transaction [%s] at [%s:%s]", tr.TxID, tmsID.Network, tmsID.Channel)
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

	logger.Infof("ownerdb [%s:%s], found [%d] pending transactions", tmsID.Network, tmsID.Channel, len(pendingTXs))

	for _, txID := range pendingTXs {
		if err := net.SubscribeTxStatusChanges(txID, &TxStatusChangesListener{net, owner.db}); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s:%s] for [%s]", tmsID.Network, tmsID.Channel, txID)
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
	auditor, err := s.(*Manager).Owner(tms.ID())
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
