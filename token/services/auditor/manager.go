/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"context"
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type TokenManagementServiceProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type StoreServiceManager db.StoreServiceManager[*auditdb.StoreService]

type TokensServiceManager db.ServiceManager[*tokens.Service]

type CheckServiceProvider interface {
	CheckService(id token.TMSID, adb *auditdb.StoreService, tdb *tokens.Service) (CheckService, error)
}

// ServiceManager handles the services
type ServiceManager struct {
	p lazy.Provider[token.TMSID, *Service]

	networkProvider     NetworkProvider
	tokenServiceManager TokensServiceManager
	tmsProvider         TokenManagementServiceProvider
}

// NewServiceManager creates a new Service manager.
func NewServiceManager(
	networkProvider NetworkProvider,
	auditStoreServiceManager StoreServiceManager,
	tokensServiceManager TokensServiceManager,
	tmsProvider TokenManagementServiceProvider,
	tracerProvider trace.TracerProvider,
	checkServiceProvider CheckServiceProvider,
) *ServiceManager {
	return &ServiceManager{
		p: lazy.NewProviderWithKeyMapper(db.Key, func(tmsID token.TMSID) (*Service, error) {
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
					Namespace:  "tokensdk",
					LabelNames: []tracing.LabelName{txIdLabel},
				})),
				checkService: checkService,
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
func (cm *ServiceManager) RestoreTMS(ctx context.Context, tmsID token.TMSID) error {
	logger.InfofContext(ctx, "restore audit dbs for entry [%s]...", tmsID)
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
	defer logger.InfofContext(ctx, "restore audit dbs for entry [%s]...done", tmsID)

	return iterators.ForEach(it, func(record *driver2.TokenRequestRecord) error {
		logger.DebugfContext(ctx, "restore transaction [%s] with status [%s]", record.TxID, TxStatusMessage[record.Status])
		return net.AddFinalityListener(tmsID.Namespace, record.TxID, common.NewFinalityListener(logger, cm.tmsProvider, tmsID, auditor.auditDB, tokenDB, auditor.finalityTracer))
	})
}

var (
	managerType = reflect.TypeOf((*ServiceManager)(nil))
)

// Get returns the Service instance for the passed auditor wallet
func Get(ctx context.Context, sp token.ServiceProvider, w *token.AuditorWallet) *Service {
	if w == nil {
		logger.DebugfContext(ctx, "no wallet provided")
		return nil
	}
	return GetByTMSID(ctx, sp, w.TMS().ID())
}

// GetByTMSID returns the Service instance for the passed auditor wallet
func GetByTMSID(ctx context.Context, sp token.ServiceProvider, tmsID token.TMSID) *Service {
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.ErrorfContext(ctx, "failed to get manager service: [%s]", err)
		return nil
	}
	auditor, err := s.(*ServiceManager).Auditor(tmsID)
	if err != nil {
		logger.ErrorfContext(ctx, "failed to get db for tms [%s]: [%s]", tmsID, err)
		return nil
	}
	return auditor
}

// New returns the Service instance for the passed auditor wallet
func New(ctx context.Context, sp token.ServiceProvider, w *token.AuditorWallet) *Service {
	return Get(ctx, sp, w)
}
