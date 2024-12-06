/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"reflect"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type TokenManagementServiceProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type AuditDBProvider interface {
	DBByTMSId(id token.TMSID) (*auditdb.DB, error)
}

type TokenDBProvider interface {
	Tokens(id token.TMSID) (*tokens.Tokens, error)
}

type CheckServiceProvider interface {
	CheckService(id token.TMSID, adb *auditdb.DB, tdb *tokens.Tokens) (CheckService, error)
}

// Manager handles the databases
type Manager struct {
	networkProvider      NetworkProvider
	auditDBProvider      AuditDBProvider
	tokenDBProvider      TokenDBProvider
	tmsProvider          TokenManagementServiceProvider
	tracerProvider       trace.TracerProvider
	checkServiceProvider CheckServiceProvider

	mutex    sync.Mutex
	auditors map[string]*Auditor
}

// NewManager creates a new Auditor manager.
func NewManager(
	networkProvider NetworkProvider,
	auditDBProvider AuditDBProvider,
	tokenDBProvider TokenDBProvider,
	tmsProvider TokenManagementServiceProvider,
	tracerProvider trace.TracerProvider,
	checkServiceProvider CheckServiceProvider,
) *Manager {
	return &Manager{
		networkProvider:      networkProvider,
		auditDBProvider:      auditDBProvider,
		tokenDBProvider:      tokenDBProvider,
		tmsProvider:          tmsProvider,
		tracerProvider:       tracerProvider,
		auditors:             map[string]*Auditor{},
		checkServiceProvider: checkServiceProvider,
	}
}

// Auditor returns the Auditor for the given wallet
func (cm *Manager) Auditor(tmsID token.TMSID) (*Auditor, error) {
	return cm.getAuditor(tmsID)
}

// RestoreTMS restores the auditdb corresponding to the passed TMS ID.
func (cm *Manager) RestoreTMS(tmsID token.TMSID) error {
	logger.Infof("restore audit dbs for entry [%s]...", tmsID)
	if err := cm.restore(tmsID); err != nil {
		return errors.Wrapf(err, "cannot bootstrap auditdb for [%s]", tmsID)
	}
	logger.Infof("restore audit dbs for entry [%s]...done", tmsID)
	return nil
}

func (cm *Manager) getAuditor(tmsID token.TMSID) (*Auditor, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := tmsID.String()
	logger.Debugf("get auditdb for [%s]", id)
	c, ok := cm.auditors[id]
	if !ok {
		var err error
		c, err = cm.newAuditor(tmsID)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to instantiate auditor for wallet [%s]", tmsID)
		}
		cm.auditors[id] = c
	}
	return c, nil
}

func (cm *Manager) newAuditor(tmsID token.TMSID) (*Auditor, error) {
	auditDB, err := cm.auditDBProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get auditdb for [%s]", tmsID)
	}
	tokenDB, err := cm.tokenDBProvider.Tokens(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get auditdb for [%s]", tmsID)
	}
	_, err = cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network instance for [%s]", tmsID)
	}
	checkService, err := cm.checkServiceProvider.CheckService(tmsID, auditDB, tokenDB)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get checkservice for [%s]", tmsID)
	}

	auditor := &Auditor{
		networkProvider: cm.networkProvider,
		tmsID:           tmsID,
		auditDB:         auditDB,
		tokenDB:         tokenDB,
		tmsProvider:     cm.tmsProvider,
		finalityTracer: cm.tracerProvider.Tracer("auditor", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "tokensdk",
			LabelNames: []tracing.LabelName{txIdLabel},
		})),
		checkService: checkService,
	}
	return auditor, nil
}

func (cm *Manager) restore(tmsID token.TMSID) error {
	net, err := cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s]", tmsID)
	}
	tokenDB, err := cm.tokenDBProvider.Tokens(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get auditdb for [%s]", tmsID)
	}
	auditor, err := cm.getAuditor(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get auditor for [%s]", tmsID)
	}
	it, err := auditor.auditDB.TokenRequests(auditdb.QueryTokenRequestsParams{Statuses: []TxStatus{auditdb.Pending}})
	if err != nil {
		return errors.Errorf("failed to get tx iterator for [%s]", tmsID)
	}
	defer it.Close()
	counter := 0
	for {
		record, err := it.Next()
		if err != nil {
			return errors.Errorf("failed to get next tx record for [%s]", tmsID)
		}
		if record == nil {
			break
		}
		logger.Debugf("restore transaction [%s] with status [%s]", record.TxID, TxStatusMessage[record.Status])
		var r driver.FinalityListener = common.NewFinalityListener(logger, cm.tmsProvider, tmsID, auditor.auditDB, tokenDB, auditor.finalityTracer)
		if err := net.AddFinalityListener(tmsID.Namespace, record.TxID, r); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s] for [%s]", tmsID, record.TokenRequest)
		}
		counter++
	}
	logger.Debugf("checked [%d] token requests", counter)
	return nil
}

var (
	managerType = reflect.TypeOf((*Manager)(nil))
)

// Get returns the Auditor instance for the passed auditor wallet
func Get(sp token.ServiceProvider, w *token.AuditorWallet) *Auditor {
	if w == nil {
		logger.Debugf("no wallet provided")
		return nil
	}
	return GetByTMSID(sp, w.TMS().ID())
}

// GetByTMSID returns the Auditor instance for the passed auditor wallet
func GetByTMSID(sp token.ServiceProvider, tmsID token.TMSID) *Auditor {
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
func New(sp token.ServiceProvider, w *token.AuditorWallet) *Auditor {
	return Get(sp, w)
}
