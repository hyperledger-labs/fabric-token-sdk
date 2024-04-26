/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/pkg/errors"
)

type TokenManagementServiceProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type AuditDBProvider interface {
	DBByTMSId(id token.TMSID) (*auditdb.DB, error)
}

// Manager handles the databases
type Manager struct {
	networkProvider NetworkProvider
	auditDBProvider AuditDBProvider

	storage  storage.DBEntriesStorage
	mutex    sync.Mutex
	auditors map[string]*Auditor
}

// NewManager creates a new Auditor manager.
func NewManager(networkProvider NetworkProvider, auditDBProvider AuditDBProvider, storage storage.DBEntriesStorage) *Manager {
	return &Manager{
		networkProvider: networkProvider,
		storage:         storage,
		auditDBProvider: auditDBProvider,
		auditors:        map[string]*Auditor{},
	}
}

// Auditor returns the Auditor for the given wallet
func (cm *Manager) Auditor(tmsID token.TMSID) (*Auditor, error) {
	return cm.getAuditor(tmsID, "")
}

func (cm *Manager) RestoreTMS(tmsID token.TMSID) error {
	logger.Infof("restore audit dbs...")
	dbs, err := cm.storage.ByTMS(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to list existing auditors")
	}
	for _, db := range dbs {
		logger.Infof("restore audit dbs for entry [%s:%s]...", db.TMSID, db.WalletID)
		if err := cm.restore(db.TMSID, db.WalletID); err != nil {
			return errors.Wrapf(err, "cannot bootstrap auditdb for [%s:%s]", db.TMSID, db.WalletID)
		}
		logger.Infof("restore audit dbs for entry [%s:%s]...done", db.TMSID, db.WalletID)
	}
	logger.Infof("restore audit dbs...done")
	return nil
}

func (cm *Manager) getAuditor(tmsID token.TMSID, walletID string) (*Auditor, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := tmsID.String() + walletID
	logger.Debugf("get auditdb for [%s]", id)
	c, ok := cm.auditors[id]
	if !ok {
		// add an entry
		if err := cm.storage.Put(tmsID, walletID); err != nil {
			return nil, errors.Wrapf(err, "failed to store auditor entry [%s:%s]", tmsID, walletID)
		}
		var err error
		c, err = cm.newAuditor(tmsID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate auditor for wallet [%s:%s]", tmsID, walletID)
		}
		cm.auditors[id] = c
	}
	return c, nil
}

func (cm *Manager) newAuditor(tmsID token.TMSID) (*Auditor, error) {
	db, err := cm.auditDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get auditdb for [%s]", tmsID)
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
	it, err := auditor.db.TokenRequests(auditdb.QueryTokenRequestsParams{Statuses: []driver.TxStatus{auditdb.Pending}})
	if err != nil {
		return errors.Errorf("failed to get tx iterator for [%s:%s]", tmsID, walletID)
	}
	defer it.Close()
	for {
		record, err := it.Next()
		if err != nil {
			return errors.Errorf("failed to get next tx record for [%s:%s]", tmsID, walletID)
		}
		if record == nil {
			break
		}
		if err := net.AddFinalityListener(record.TxID, &FinalityListener{net, auditor.db}); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s] for [%s]", tmsID, record.TokenRequest)
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
	return GetByTMSID(sp, w.TMS().ID())
}

// GetByTMSID returns the Auditor instance for the passed auditor wallet
func GetByTMSID(sp view.ServiceProvider, tmsID token.TMSID) *Auditor {
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.Errorf("failed to get manager service: [%s]", err)
		return nil
	}
	auditor, err := s.(*Manager).Auditor(tmsID)
	if err != nil {
		logger.Errorf("failed to get db for tms [%s]: [%s]", tmsID, err)
		return nil
	}
	return auditor
}

// New returns the Auditor instance for the passed auditor wallet
func New(sp view.ServiceProvider, w *token.AuditorWallet) *Auditor {
	return Get(sp, w)
}
