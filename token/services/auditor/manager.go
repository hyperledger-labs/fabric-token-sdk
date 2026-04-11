/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"context"
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/finality"
	"go.opentelemetry.io/otel/trace"
)

type TokenManagementServiceProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

// auditTokenExistenceChecker can tell whether a given txID was already committed to tokenDB.
type auditTokenExistenceChecker interface {
	TransactionExists(ctx context.Context, id string) (bool, error)
}

// auditStatusSetter can promote an auditDB record from Pending to Confirmed.
type auditStatusSetter interface {
	SetStatus(ctx context.Context, txID string, status TxStatus, message string) error
}

// recoverAuditCommittedPending heals the split-brain state that arises when the node crashes
// after auditDB.Append (tokenDB write) succeeds but before auditDB.SetStatus(Confirmed) runs.
//
// Returns true if the record was healed — the caller should then skip AddFinalityListener
// because the transaction is already fully committed.
// Returns false on any error so the caller falls back to the normal finality-listener path.
func recoverAuditCommittedPending(ctx context.Context, txID string, checker auditTokenExistenceChecker, setter auditStatusSetter) bool {
	committed, err := checker.TransactionExists(ctx, txID)
	if err != nil {
		logger.Warnf("recover audit tx [%s]: failed to check token existence, falling back to finality listener: %v", txID, err)

		return false
	}
	if !committed {
		return false
	}
	logger.Infof("recover audit tx [%s]: tokens committed to tokenDB but auditDB still Pending; setting Confirmed directly", txID)
	if err := setter.SetStatus(ctx, txID, auditdb.Confirmed, "recovered on restart: tokenDB committed before auditDB status update"); err != nil {
		logger.Errorf("recover audit tx [%s]: failed to set Confirmed: %v; falling back to finality listener", txID, err)

		return false
	}

	return true
}

type StoreServiceManager = auditdb.StoreServiceManager

type TokensServiceManager = services.ServiceManager[*tokens.Service]

type CheckServiceProvider interface {
	CheckService(id token.TMSID, adb *auditdb.StoreService, tdb *tokens.Service) (CheckService, error)
}

// ServiceManager handles the services
type ServiceManager struct {
	p lazy.Provider[token.TMSID, *Service]

	networkProvider     NetworkProvider
	tokenServiceManager TokensServiceManager
	tmsProvider         dep.TokenManagementServiceProvider
}

// NewServiceManager creates a new Service manager.
func NewServiceManager(
	networkProvider NetworkProvider,
	auditStoreServiceManager StoreServiceManager,
	tokensServiceManager TokensServiceManager,
	tmsProvider dep.TokenManagementServiceProvider,
	tracerProvider trace.TracerProvider,
	metricsProvider metrics.Provider,
	checkServiceProvider CheckServiceProvider,
) *ServiceManager {
	return &ServiceManager{
		p: lazy.NewProviderWithKeyMapper(services.Key, func(tmsID token.TMSID) (*Service, error) {
			auditDB, err := auditStoreServiceManager.StoreServiceByTMSId(tmsID)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get auditdb for [%s]", tmsID)
			}
			tokenDB, err := tokensServiceManager.ServiceByTMSId(tmsID)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get auditdb for [%s]", tmsID)
			}
			_, err = networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get network instance for [%s]", tmsID)
			}
			checkService, err := checkServiceProvider.CheckService(tmsID, auditDB, tokenDB)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get checkservice for [%s]", tmsID)
			}

			auditor := &Service{
				networkProvider: networkProvider,
				tmsID:           tmsID,
				auditDB:         auditDB,
				tokenDB:         tokenDB,
				tmsProvider:     tmsProvider,
				finalityTracer: tracerProvider.Tracer("auditor", tracing.WithMetricsOpts(tracing.MetricsOpts{
					LabelNames: []tracing.LabelName{txIdLabel},
				})),
				metricsProvider: metricsProvider,
				metrics:         newMetrics(metricsProvider),
				checkService:    checkService,
			}

			return auditor, nil
		}),
		networkProvider:     networkProvider,
		tokenServiceManager: tokensServiceManager,
		tmsProvider:         tmsProvider,
	}
}

// Auditor returns the Service for the given wallet
func (cm *ServiceManager) Auditor(tmsID token.TMSID) (*Service, error) {
	return cm.p.Get(tmsID)
}

// RestoreTMS restores the auditdb corresponding to the passed TMS ID.
func (cm *ServiceManager) RestoreTMS(tmsID token.TMSID) error {
	logger.Infof("restore audit dbs for entry [%s]...", tmsID)
	net, err := cm.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s]", tmsID)
	}
	tokenDB, err := cm.tokenServiceManager.ServiceByTMSId(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get auditdb for [%s]", tmsID)
	}
	auditor, err := cm.p.Get(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get auditor for [%s]", tmsID)
	}
	it, err := auditor.auditDB.TokenRequests(context.Background(), auditdb.QueryTokenRequestsParams{Statuses: []TxStatus{auditdb.Pending}})
	if err != nil {
		return errors.Errorf("failed to get tx iterator for [%s]", tmsID)
	}
	defer logger.Infof("restore audit dbs for entry [%s]...done", tmsID)

	return iterators.ForEach(it, func(record *storage.TokenRequestRecord) error {
		logger.Debugf("restore transaction [%s] with status [%s]", record.TxID, TxStatusMessage[record.Status])

		// Crash-recovery: if tokens were already committed to tokenDB but the audit
		// status was never flipped to Confirmed (crash between Append and SetStatus),
		// heal the record directly instead of waiting for a finality event that may
		// never be re-delivered.
		if recoverAuditCommittedPending(context.Background(), record.TxID, tokenDB.Storage, auditor.auditDB) {
			return nil
		}

		return net.AddFinalityListener(
			tmsID.Namespace,
			record.TxID,
			finality.NewListener(
				logger,
				net,
				tmsID.Namespace,
				cm.tmsProvider,
				tmsID,
				auditor.auditDB,
				tokenDB,
				auditor.finalityTracer,
				auditor.metricsProvider,
			),
		)
	})
}

var (
	managerType = reflect.TypeOf((*ServiceManager)(nil))
)

// Get returns the Service instance for the passed auditor wallet
func Get(sp token.ServiceProvider, w *token.AuditorWallet) *Service {
	if w == nil {
		logger.Debugf("no wallet provided")

		return nil
	}

	return GetByTMSID(sp, w.TMS().ID())
}

// GetByTMSID returns the Service instance for the passed auditor wallet
func GetByTMSID(sp token.ServiceProvider, tmsID token.TMSID) *Service {
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.Errorf("failed to get manager service: [%s]", err)

		return nil
	}
	auditor, err := s.(*ServiceManager).Auditor(tmsID)
	if err != nil {
		logger.Errorf("failed to get db for tms [%s]: [%s]", tmsID, err)

		return nil
	}

	return auditor
}
