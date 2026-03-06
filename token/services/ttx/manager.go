/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/finality"
	"go.opentelemetry.io/otel/trace"
)

// StoreServiceManager handles StoreService instances.
type StoreServiceManager = ttxdb.StoreServiceManager

// TokensServiceManager handles TokensService instances.
type TokensServiceManager = dep.TokensServiceManager

// CheckServiceProvider handles CheckService instances.
type CheckServiceProvider = dep.CheckServiceProvider

// ServiceManager handles the lifecycle of Service instances per TMS.
type ServiceManager struct {
	p lazy.Provider[token.TMSID, *Service]

	networkProvider      dep.NetworkProvider
	tokensServiceManager TokensServiceManager
}

// NewServiceManager creates a new ServiceManager.
func NewServiceManager(
	networkProvider dep.NetworkProvider,
	tmsProvider dep.TokenManagementServiceProvider,
	ttxStoreServiceManager dep.StoreServiceManager,
	tokensServiceManager TokensServiceManager,
	tracerProvider trace.TracerProvider,
	checkServiceProvider CheckServiceProvider,
) *ServiceManager {
	return &ServiceManager{
		p: lazy.NewProviderWithKeyMapper(services.Key, func(tmsID token.TMSID) (*Service, error) {
			ttxStoreService, err := ttxStoreServiceManager.StoreServiceByTMSId(tmsID)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s]", tmsID)
			}
			tokensService, err := tokensServiceManager.ServiceByTMSId(tmsID)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get ttxdb for [%s]", tmsID)
			}
			checkService, err := checkServiceProvider.CheckService(tmsID, ttxStoreService, tokensService)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get checkservice for [%s]", tmsID)
			}
			wrapper := &Service{
				networkProvider: networkProvider,
				tmsID:           tmsID,
				tmsProvider:     tmsProvider,
				ttxStoreService: ttxStoreService,
				tokensService:   tokensService,
				finalityTracer: tracerProvider.Tracer("db", tracing.WithMetricsOpts(tracing.MetricsOpts{
					LabelNames: []tracing.LabelName{txIdLabel},
				})),
				checkService: checkService,
			}
			_, err = networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
			}

			return wrapper, nil
		}),
		networkProvider:      networkProvider,
		tokensServiceManager: tokensServiceManager,
	}
}

// ServiceByTMSId returns the Service for the given TMS ID.
func (m *ServiceManager) ServiceByTMSId(tmsID token.TMSID) (*Service, error) {
	return m.p.Get(tmsID)
}

// RestoreTMS restores the state for the passed TMS ID by re-subscribing to finality for pending transactions.
func (m *ServiceManager) RestoreTMS(ctx context.Context, tmsID token.TMSID) error {
	net, err := m.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
	}

	db, err := m.ServiceByTMSId(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get db for [%s:%s]", tmsID.Network, tmsID.Channel)
	}

	it, err := db.ttxStoreService.TokenRequests(ctx, ttxdb.QueryTokenRequestsParams{Statuses: []TxStatus{storage.Pending}})
	if err != nil {
		return errors.WithMessagef(err, "failed to get tx iterator for [%s:%s:%s]", tmsID.Network, tmsID.Channel, tmsID)
	}

	return iterators.ForEach(it, func(record *storage.TokenRequestRecord) error {
		logger.Debugf("restore transaction [%s] with status [%s]", record.TxID, TxStatusMessage[record.Status])

		return net.AddFinalityListener(
			tmsID.Namespace,
			record.TxID,
			finality.NewListener(
				logger,
				net,
				tmsID.Namespace,
				db.tmsProvider,
				db.tmsID,
				db.ttxStoreService,
				db.tokensService,
				db.finalityTracer,
			),
		)
	})
}

// CacheRequest stores the request's details for later use.
func (m *ServiceManager) CacheRequest(ctx context.Context, tmsID token.TMSID, request *token.Request) error {
	service, err := m.tokensServiceManager.ServiceByTMSId(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get service for [%s]", tmsID)
	}

	return service.CacheRequest(ctx, tmsID, request)
}

var (
	managerType = reflect.TypeOf((*ServiceManager)(nil))
)

// Get returns the Service instance for the passed TMS.
func Get(sp token.ServiceProvider, tms dep.TokenManagementService) *Service {
	if tms == nil {
		logger.Debugf("no TMS provided")

		return nil
	}
	s, err := sp.GetService(managerType)
	if err != nil {
		logger.Errorf("failed to get manager service: [%s]", err)

		return nil
	}
	auditor, err := s.(*ServiceManager).ServiceByTMSId(tms.ID())
	if err != nil {
		logger.Errorf("failed to get db for TMS [%s]: [%s]", tms.ID(), err)

		return nil
	}

	return auditor
}
