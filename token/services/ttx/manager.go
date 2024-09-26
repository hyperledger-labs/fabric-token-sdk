/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type DBProvider interface {
	DBByTMSId(id token.TMSID) (*ttxdb.DB, error)
}

type TokensProvider interface {
	Tokens(tmsID token.TMSID) (*tokens.Tokens, error)
}

type TMSProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

// Manager handles the databases
type Manager struct {
	networkProvider NetworkProvider
	tmsProvider     TMSProvider
	ttxDBProvider   DBProvider
	tokensProvider  TokensProvider
	tracerProvider  trace.TracerProvider

	mutex sync.Mutex
	dbs   map[string]*DB
}

// NewManager creates a new DB manager.
func NewManager(
	np NetworkProvider,
	tmsProvider TMSProvider,
	ttxDBProvider DBProvider,
	tokensBProvider TokensProvider,
	tracerProvider trace.TracerProvider,
) *Manager {
	return &Manager{
		networkProvider: np,
		tmsProvider:     tmsProvider,
		ttxDBProvider:   ttxDBProvider,
		tokensProvider:  tokensBProvider,
		tracerProvider:  tracerProvider,
		dbs:             map[string]*DB{},
	}
}

// DB returns the DB for the given TMS
func (m *Manager) DB(tmsID token.TMSID) (*DB, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	id := tmsID.String()
	logger.Debugf("get ttxdb for [%s]", id)
	c, ok := m.dbs[id]
	if !ok {
		var err error
		c, err = m.newDB(tmsID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate db for TMS [%s]", tmsID)
		}
		m.dbs[id] = c
	}
	return c, nil
}

func (m *Manager) newDB(tmsID token.TMSID) (*DB, error) {
	ttxDB, err := m.ttxDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s]", tmsID)
	}
	tokenDB, err := m.tokensProvider.Tokens(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s]", tmsID)
	}
	wrapper := &DB{
		networkProvider: m.networkProvider,
		tmsID:           tmsID,
		tmsProvider:     m.tmsProvider,
		ttxDB:           ttxDB,
		tokenDB:         tokenDB,
		finalityTracer: m.tracerProvider.Tracer("db", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "tokensdk",
			LabelNames: []tracing.LabelName{txIdLabel},
		})),
	}
	_, err = m.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
	}
	return wrapper, nil
}

// RestoreTMS restores the auditdb corresponding to the passed TMS ID.
func (m *Manager) RestoreTMS(tmsID token.TMSID) error {
	net, err := m.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
	}

	db, err := m.DB(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get db for [%s:%s]", tmsID.Network, tmsID.Channel)
	}

	it, err := db.ttxDB.TokenRequests(ttxdb.QueryTokenRequestsParams{Statuses: []TxStatus{driver.Pending}})
	if err != nil {
		return errors.WithMessagef(err, "failed to get tx iterator for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID)
	}
	defer it.Close()
	counter := 0
	for {
		record, err := it.Next()
		if err != nil {
			return errors.WithMessagef(err, "failed to get next tx record for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID)
		}
		if record == nil {
			break
		}
		logger.Debugf("restore transaction [%s] with status [%s]", record.TxID, TxStatusMessage[record.Status])
		if err := net.AddFinalityListener(tmsID.Namespace, record.TxID, common.NewFinalityListener(logger, db.tmsProvider, db.tmsID, db.ttxDB, db.tokenDB, db.finalityTracer)); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s:%s] for [%s]", tmsID.Network, tmsID.Channel, record.TxID)
		}
		counter++
	}
	logger.Debugf("checked [%d] token requests", counter)
	return nil
}

var (
	managerType = reflect.TypeOf((*Manager)(nil))
)

// Get returns the DB instance for the passed TMS
func Get(sp token.ServiceProvider, tms *token.ManagementService) *DB {
	if tms == nil {
		logger.Debugf("no TMS provided")
		return nil
	}
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.Errorf("failed to get manager service: [%s]", err)
		return nil
	}
	auditor, err := s.(*Manager).DB(tms.ID())
	if err != nil {
		logger.Errorf("failed to get db for TMS [%s]: [%s]", tms.ID(), err)
		return nil
	}
	return auditor
}

// New returns the DB instance for the passed TMS
func New(sp token.ServiceProvider, tms *token.ManagementService) *DB {
	return Get(sp, tms)
}
