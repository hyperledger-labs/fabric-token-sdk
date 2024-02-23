/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/pkg/errors"
)

type DBProvider interface {
	DBByTMSId(id token.TMSID) (*tokendb.DB, error)
}

// Manager handles the databases
type Manager struct {
	networkProvider NetworkProvider
	dbProvider      DBProvider

	storage storage.DBEntriesStorage
	mutex   sync.Mutex
	owners  map[string]*Tokens
}

// NewManager creates a new Tokens manager.
func NewManager(np NetworkProvider, ttxdbManager DBProvider, storage storage.DBEntriesStorage) *Manager {
	return &Manager{
		networkProvider: np,
		storage:         storage,
		dbProvider:      ttxdbManager,
		owners:          map[string]*Tokens{},
	}
}

// Tokens returns the Tokens for the given TMS
func (cm *Manager) Tokens(tmsID token.TMSID) (*Tokens, error) {
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
		c, err = cm.newTokens(tmsID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate owner for wallet [%s]", tmsID)
		}
		cm.owners[id] = c
	}
	return c, nil
}

func (cm *Manager) newTokens(tmsID token.TMSID) (*Tokens, error) {
	db, err := cm.dbProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s]", tmsID)
	}
	owner := &Tokens{
		networkProvider: cm.networkProvider,
		db:              db,
	}
	_, err = cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
	}
	return owner, nil
}

var (
	managerType = reflect.TypeOf((*Manager)(nil))
)

// Get returns the Tokens instance for the passed TMS
func Get(sp view.ServiceProvider, tms *token.ManagementService) *Tokens {
	if tms == nil {
		logger.Debugf("no TMS provided")
		return nil
	}
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.Errorf("failed to get manager service: [%s]", err)
		return nil
	}
	auditor, err := s.(*Manager).Tokens(tms.ID())
	if err != nil {
		logger.Errorf("failed to get db for TMS [%s]: [%s]", tms.ID(), err)
		return nil
	}
	return auditor
}

// New returns the Tokens instance for the passed TMS
func New(sp view.ServiceProvider, tms *token.ManagementService) *Tokens {
	return Get(sp, tms)
}
