/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/pkg/errors"
)

type DBProvider interface {
	DBByTMSId(id token.TMSID) (*tokendb.DB, error)
}

type TMSProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

// Manager handles the databases
type Manager struct {
	tmsProvider   TMSProvider
	dbProvider    DBProvider
	notifier      events.Publisher
	authorization Authorization
	issued        Issued
	storage       storage.DBEntriesStorage

	mutex  sync.Mutex
	tokens map[string]*Tokens
}

// NewManager creates a new Tokens manager.
func NewManager(
	tmsProvider TMSProvider,
	dbManager DBProvider,
	notifier events.Publisher,
	authorization Authorization,
	issued Issued,
	storage storage.DBEntriesStorage,
) *Manager {
	return &Manager{
		tmsProvider:   tmsProvider,
		dbProvider:    dbManager,
		notifier:      notifier,
		authorization: authorization,
		issued:        issued,
		storage:       storage,
		tokens:        map[string]*Tokens{},
	}
}

// Tokens returns the Tokens for the given TMS
func (cm *Manager) Tokens(tmsID token.TMSID) (*Tokens, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := tmsID.String()
	logger.Debugf("get ttxdb for [%s]", id)
	c, ok := cm.tokens[id]
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
		cm.tokens[id] = c
	}
	return c, nil
}

func (cm *Manager) newTokens(tmsID token.TMSID) (*Tokens, error) {
	db, err := cm.dbProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get tokendb for [%s]", tmsID)
	}

	storage, err := NewDBStorage(cm.notifier, db, tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token store for [%s]", tmsID)
	}
	tokens := &Tokens{
		TMSProvider: cm.tmsProvider,
		Ownership:   cm.authorization,
		Issued:      cm.issued,
		Storage:     storage,
	}
	return tokens, nil
}

var (
	managerType = reflect.TypeOf((*Manager)(nil))
)

// Get returns the Tokens instance for the passed TMS
func Get(sp view.ServiceProvider, tmsID token.TMSID) (*Tokens, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, err
	}
	tokens, err := s.(*Manager).Tokens(tmsID)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}
