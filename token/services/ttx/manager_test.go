/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// MockServiceProvider is a mock for token.ServiceProvider.
// It allows registering and retrieving services by their reflect.Type.
type MockServiceProvider struct {
	services map[reflect.Type]interface{}
}

// GetService retrieves a service by its type.
func (m *MockServiceProvider) GetService(v interface{}) (interface{}, error) {
	typ := v.(reflect.Type)
	s, ok := m.services[typ]
	if !ok {
		return nil, errors.Errorf("service not found")
	}

	return s, nil
}

// TokenRequestIteratorMock is a mock for TokenRequestIterator.
// It allows defining custom Next and Close behavior for tests.
type TokenRequestIteratorMock struct {
	nextFunc  func() (*storage.TokenRequestRecord, error)
	closeFunc func()
}

// Next calls the mocked nextFunc.
func (m *TokenRequestIteratorMock) Next() (*storage.TokenRequestRecord, error) {
	return m.nextFunc()
}

// Close calls the mocked closeFunc if it is defined.
func (m *TokenRequestIteratorMock) Close() {
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

func TestServiceManager(t *testing.T) {
	tmsID := token.TMSID{Network: "n1", Channel: "c1", Namespace: "ns1"}

	setup := func() (
		*mock.NetworkProvider,
		*mock.TokenManagementServiceProvider,
		*mock.StoreServiceManager,
		*mock.TokensServiceManager,
		trace.TracerProvider,
		*mock.CheckServiceProvider,
		*ttx.ServiceManager,
	) {
		networkProvider := &mock.NetworkProvider{}
		tmsProvider := &mock.TokenManagementServiceProvider{}
		ttxStoreServiceManager := &mock.StoreServiceManager{}
		tokensServiceManager := &mock.TokensServiceManager{}
		tracerProvider := noop.NewTracerProvider()
		checkServiceProvider := &mock.CheckServiceProvider{}

		manager := ttx.NewServiceManager(
			networkProvider,
			tmsProvider,
			ttxStoreServiceManager,
			tokensServiceManager,
			tracerProvider,
			checkServiceProvider,
		)

		return networkProvider, tmsProvider, ttxStoreServiceManager, tokensServiceManager, tracerProvider, checkServiceProvider, manager
	}

	t.Run("ServiceByTMSId Success", func(t *testing.T) {
		networkProvider, _, ttxStoreServiceManager, tokensServiceManager, _, checkServiceProvider, manager := setup()

		ttxStoreService := &mock.StoreService{}
		tokensService := &mock.TokensService{}
		checkService := &mock.CheckService{}
		network := &mock.Network{}

		ttxStoreServiceManager.StoreServiceByTMSIdReturns(ttxStoreService, nil)
		tokensServiceManager.ServiceByTMSIdReturns(tokensService, nil)
		checkServiceProvider.CheckServiceReturns(checkService, nil)
		networkProvider.GetNetworkReturns(network, nil)

		service, err := manager.ServiceByTMSId(tmsID)
		require.NoError(t, err)
		assert.NotNil(t, service)

		// Check calls
		assert.Equal(t, 1, ttxStoreServiceManager.StoreServiceByTMSIdCallCount())
		assert.Equal(t, tmsID, ttxStoreServiceManager.StoreServiceByTMSIdArgsForCall(0))
	})

	t.Run("ServiceByTMSId Failure StoreService", func(t *testing.T) {
		_, _, ttxStoreServiceManager, _, _, _, manager := setup()
		tmsID2 := token.TMSID{Network: "n2"}
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(nil, errors.New("store error"))

		service, err := manager.ServiceByTMSId(tmsID2)
		require.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "failed to get ttxdb")
		assert.Contains(t, err.Error(), "store error")
	})

	t.Run("ServiceByTMSId Failure TokensService", func(t *testing.T) {
		_, _, ttxStoreServiceManager, tokensServiceManager, _, _, manager := setup()
		tmsID3 := token.TMSID{Network: "n3"}
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(&mock.StoreService{}, nil)
		tokensServiceManager.ServiceByTMSIdReturns(nil, errors.New("tokens error"))

		service, err := manager.ServiceByTMSId(tmsID3)
		require.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "failed to get ttxdb")
		assert.Contains(t, err.Error(), "tokens error")
	})

	t.Run("RestoreTMS Success", func(t *testing.T) {
		networkProvider, _, ttxStoreServiceManager, tokensServiceManager, _, checkServiceProvider, manager := setup()

		network := &mock.Network{}
		networkProvider.GetNetworkReturns(network, nil)

		ttxStoreService := &mock.StoreService{}
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(ttxStoreService, nil)

		tokensService := &mock.TokensService{}
		tokensServiceManager.ServiceByTMSIdReturns(tokensService, nil)

		checkService := &mock.CheckService{}
		checkServiceProvider.CheckServiceReturns(checkService, nil)

		// Mock iterator
		txID := "tx1"
		it := &TokenRequestIteratorMock{
			nextFunc: func() (*storage.TokenRequestRecord, error) {
				if txID == "" {
					return nil, nil // End of iterator
				}
				res := &storage.TokenRequestRecord{TxID: txID, Status: storage.Pending}
				txID = ""

				return res, nil
			},
		}
		ttxStoreService.TokenRequestsReturns(it, nil)

		err := manager.RestoreTMS(context.Background(), tmsID)
		require.NoError(t, err)

		assert.Equal(t, 1, network.AddFinalityListenerCallCount())
	})

	t.Run("RestoreTMS Failure Network", func(t *testing.T) {
		networkProvider, _, _, _, _, _, manager := setup()
		networkProvider.GetNetworkReturns(nil, errors.New("network error"))
		err := manager.RestoreTMS(context.Background(), tmsID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get network instance")
	})

	t.Run("CacheRequest Success", func(t *testing.T) {
		_, _, _, tokensServiceManager, _, _, manager := setup()
		tokensService := &mock.TokensService{}
		tokensServiceManager.ServiceByTMSIdReturns(tokensService, nil)
		tokensService.CacheRequestReturns(nil)

		err := manager.CacheRequest(context.Background(), tmsID, &token.Request{})
		require.NoError(t, err)
		assert.Equal(t, 1, tokensService.CacheRequestCallCount())
	})

	t.Run("Get Success", func(t *testing.T) {
		networkProvider, _, ttxStoreServiceManager, tokensServiceManager, _, checkServiceProvider, manager := setup()
		sp := &MockServiceProvider{
			services: make(map[reflect.Type]interface{}),
		}
		sp.services[reflect.TypeOf(manager)] = manager

		tms := &mock.TokenManagementService{}
		tms.IDReturns(tmsID)

		// Setup manager to return a service for tmsID
		ttxStoreService := &mock.StoreService{}
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(ttxStoreService, nil)
		tokensService := &mock.TokensService{}
		tokensServiceManager.ServiceByTMSIdReturns(tokensService, nil)
		checkService := &mock.CheckService{}
		checkServiceProvider.CheckServiceReturns(checkService, nil)
		network := &mock.Network{}
		networkProvider.GetNetworkReturns(network, nil)

		service := ttx.Get(sp, tms)
		assert.NotNil(t, service)
	})

	t.Run("ServiceByTMSId Failure CheckService", func(t *testing.T) {
		_, _, ttxStoreServiceManager, tokensServiceManager, _, checkServiceProvider, manager := setup()
		tmsID4 := token.TMSID{Network: "n4"}
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(&mock.StoreService{}, nil)
		tokensServiceManager.ServiceByTMSIdReturns(&mock.TokensService{}, nil)
		checkServiceProvider.CheckServiceReturns(nil, errors.New("check error"))

		service, err := manager.ServiceByTMSId(tmsID4)
		require.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "failed to get checkservice")
		assert.Contains(t, err.Error(), "check error")
	})

	t.Run("ServiceByTMSId Failure Network", func(t *testing.T) {
		networkProvider, _, ttxStoreServiceManager, tokensServiceManager, _, checkServiceProvider, manager := setup()
		tmsID5 := token.TMSID{Network: "n5"}
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(&mock.StoreService{}, nil)
		tokensServiceManager.ServiceByTMSIdReturns(&mock.TokensService{}, nil)
		checkServiceProvider.CheckServiceReturns(&mock.CheckService{}, nil)
		networkProvider.GetNetworkReturns(nil, errors.New("network error"))

		service, err := manager.ServiceByTMSId(tmsID5)
		require.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "failed to get network instance")
		assert.Contains(t, err.Error(), "network error")
	})

	t.Run("RestoreTMS Failure ServiceByTMSId", func(t *testing.T) {
		networkProvider, _, ttxStoreServiceManager, _, _, _, manager := setup()
		networkProvider.GetNetworkReturns(&mock.Network{}, nil)
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(nil, errors.New("store error"))

		err := manager.RestoreTMS(context.Background(), tmsID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get db")
	})

	t.Run("RestoreTMS Failure TokenRequests", func(t *testing.T) {
		networkProvider, _, ttxStoreServiceManager, tokensServiceManager, _, checkServiceProvider, manager := setup()

		networkProvider.GetNetworkReturns(&mock.Network{}, nil)
		ttxStoreService := &mock.StoreService{}
		ttxStoreServiceManager.StoreServiceByTMSIdReturns(ttxStoreService, nil)
		tokensServiceManager.ServiceByTMSIdReturns(&mock.TokensService{}, nil)
		checkServiceProvider.CheckServiceReturns(&mock.CheckService{}, nil)

		ttxStoreService.TokenRequestsReturns(nil, errors.New("iterator error"))

		err := manager.RestoreTMS(context.Background(), tmsID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get tx iterator")
		assert.Contains(t, err.Error(), "iterator error")
	})

	t.Run("CacheRequest Failure TokensService", func(t *testing.T) {
		_, _, _, tokensServiceManager, _, _, manager := setup()
		tokensServiceManager.ServiceByTMSIdReturns(nil, errors.New("tokens error"))

		err := manager.CacheRequest(context.Background(), tmsID, &token.Request{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get service")
	})

	t.Run("Get Nil TMS", func(t *testing.T) {
		_, _, _, _, _, _, _ = setup()
		sp := &MockServiceProvider{}
		service := ttx.Get(sp, nil)
		assert.Nil(t, service)
	})
}
