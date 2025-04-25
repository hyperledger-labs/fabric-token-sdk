/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type StoreServiceManager db.StoreServiceManager[*ttxdb.StoreService]

type TokensServiceManager db.ServiceManager[*tokens.Service]

type TMSProvider interface {
	GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error)
}

type CheckServiceProvider interface {
	CheckService(id token.TMSID, adb *ttxdb.StoreService, tdb *tokens.Service) (CheckService, error)
}

// ServiceManager handles the services
type ServiceManager struct {
	p lazy.Provider[token.TMSID, *Service]

	networkProvider NetworkProvider
}

// NewServiceManager creates a new Service manager.
func NewServiceManager(
	networkProvider NetworkProvider,
	tmsProvider TMSProvider,
	ttxStoreServiceManager StoreServiceManager,
	tokensServiceManager TokensServiceManager,
	tracerProvider trace.TracerProvider,
	checkServiceProvider CheckServiceProvider,
) *ServiceManager {
	return &ServiceManager{
		p: lazy.NewProviderWithKeyMapper(db.Key, func(tmsID token.TMSID) (*Service, error) {
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
					Namespace:  "tokensdk",
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
		networkProvider: networkProvider,
	}
}

// ServiceByTMSId returns the Service for the given TMS
func (m *ServiceManager) ServiceByTMSId(tmsID token.TMSID) (*Service, error) {
	return m.p.Get(tmsID)
}

// RestoreTMS restores the ttxdb corresponding to the passed TMS ID.
func (m *ServiceManager) RestoreTMS(tmsID token.TMSID) error {
	net, err := m.networkProvider.GetNetwork(tmsID.Network, tmsID.Channel)
	if err != nil {
		return errors.WithMessagef(err, "failed to get network instance for [%s:%s]", tmsID.Network, tmsID.Channel)
	}

	db, err := m.ServiceByTMSId(tmsID)
	if err != nil {
		return errors.WithMessagef(err, "failed to get db for [%s:%s]", tmsID.Network, tmsID.Channel)
	}

	it, err := db.ttxStoreService.TokenRequests(ttxdb.QueryTokenRequestsParams{Statuses: []TxStatus{driver.Pending}})
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
		if err := net.AddFinalityListener(tmsID.Namespace, record.TxID, common.NewFinalityListener(logger, db.tmsProvider, db.tmsID, db.ttxStoreService, db.tokensService, db.finalityTracer)); err != nil {
			return errors.WithMessagef(err, "failed to subscribe event listener to network [%s:%s] for [%s]", tmsID.Network, tmsID.Channel, record.TxID)
		}
		counter++
	}
	logger.Debugf("checked [%d] token requests", counter)
	return nil
}

var (
	managerType = reflect.TypeOf((*ServiceManager)(nil))
)

// Get returns the Service instance for the passed TMS
func Get(sp token.ServiceProvider, tms *token.ManagementService) *Service {
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

// New returns the Service instance for the passed TMS
func New(sp token.ServiceProvider, tms *token.ManagementService) *Service {
	return Get(sp, tms)
}
