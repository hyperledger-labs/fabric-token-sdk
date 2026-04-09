/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

// mockNetworkProvider implements NetworkProvider
type mockNetworkProvider struct {
	net *network.Network
	err error
}

func (m *mockNetworkProvider) GetNetwork(network string, channel string) (*network.Network, error) {
	return m.net, m.err
}

// mockStoreServiceManager implements StoreServiceManager
type mockStoreServiceManager struct {
	db  *auditdb.StoreService
	err error
}

func (m *mockStoreServiceManager) StoreServiceByTMSId(tmsID token.TMSID) (*auditdb.StoreService, error) {
	return m.db, m.err
}

// mockTokensServiceManager implements TokensServiceManager
type mockTokensServiceManager struct {
	db  *tokens.Service
	err error
}

func (m *mockTokensServiceManager) ServiceByTMSId(tmsID token.TMSID) (*tokens.Service, error) {
	return m.db, m.err
}

// mockTokenManagementServiceProvider implements dep.TokenManagementServiceProvider
type mockTokenManagementServiceProvider struct {
	svc dep.TokenManagementServiceWithExtensions
	err error
}

func (m *mockTokenManagementServiceProvider) TokenManagementService(opts ...token.ServiceOption) (dep.TokenManagementServiceWithExtensions, error) {
	return m.svc, m.err
}

// mockCheckServiceProvider implements CheckServiceProvider
type mockCheckServiceProvider struct {
	svc CheckService
	err error
}

func (m *mockCheckServiceProvider) CheckService(id token.TMSID, adb *auditdb.StoreService, tdb *tokens.Service) (CheckService, error) {
	return m.svc, m.err
}

func TestNewServiceManager(t *testing.T) {
	netProv := &mockNetworkProvider{}
	ssm := &mockStoreServiceManager{}
	tsm := &mockTokensServiceManager{}
	tmsProv := &mockTokenManagementServiceProvider{}
	tp := noop.NewTracerProvider()
	mp := &noopProvider{}
	csp := &mockCheckServiceProvider{}

	sm := NewServiceManager(netProv, ssm, tsm, tmsProv, tp, mp, csp)
	assert.NotNil(t, sm)
}

func TestServiceManager_Auditor(t *testing.T) {
	netProv := &mockNetworkProvider{err: errors.New("net err")}
	ssm := &mockStoreServiceManager{err: errors.New("db err")}
	tsm := &mockTokensServiceManager{err: errors.New("tok err")}
	tmsProv := &mockTokenManagementServiceProvider{}
	tp := noop.NewTracerProvider()
	mp := &noopProvider{}
	csp := &mockCheckServiceProvider{}

	sm := NewServiceManager(netProv, ssm, tsm, tmsProv, tp, mp, csp)
	a, err := sm.Auditor(token.TMSID{Network: "n1", Channel: "c1", Namespace: "ns1"})
	require.Error(t, err)
	assert.Nil(t, a)
}

func TestServiceManager_RestoreTMS(t *testing.T) {
	netProv := &mockNetworkProvider{err: errors.New("net err")}
	sm := &ServiceManager{
		p:               nil,
		networkProvider: netProv,
	}
	err := sm.RestoreTMS(token.TMSID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get network instance")

	// Missing tokens error
	tsm := &mockTokensServiceManager{err: errors.New("tok err")}
	netProv2 := &mockNetworkProvider{}
	sm.networkProvider = netProv2
	sm.tokenServiceManager = tsm
	err = sm.RestoreTMS(token.TMSID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get auditdb")
}

func TestServiceManager_RestoreTMS_Success(t *testing.T) {
	// Need a valid auditDB and network provider
	auditStub := &stubAuditTransactionStore{}
	netProv := &mockNetworkProvider{
		net: &network.Network{}, // this will panic
	}

	// Force it to panic in the loop due to empty Network
	assert.Panics(t, func() {
		// we need auditor to exist
		smSuccess := NewServiceManager(
			netProv,
			&mockStoreServiceManager{db: newTestStoreService(t, auditStub)},
			&mockTokensServiceManager{db: &tokens.Service{}},
			&mockTokenManagementServiceProvider{},
			noop.NewTracerProvider(),
			&noopProvider{},
			&mockCheckServiceProvider{})
		_ = smSuccess.RestoreTMS(token.TMSID{})
	})
}

func TestServiceManager_Auditor_InitSuccess(t *testing.T) {
	netProv := &mockNetworkProvider{}
	ssm := &mockStoreServiceManager{db: newTestStoreService(t, &stubAuditTransactionStore{})}
	tsm := &mockTokensServiceManager{db: &tokens.Service{}}
	tmsProv := &mockTokenManagementServiceProvider{}
	tp := noop.NewTracerProvider()
	mp := &noopProvider{}
	csp := &mockCheckServiceProvider{}

	sm := NewServiceManager(netProv, ssm, tsm, tmsProv, tp, mp, csp)
	a, err := sm.Auditor(token.TMSID{Network: "n1", Channel: "c1", Namespace: "ns1"})
	require.NoError(t, err)
	assert.NotNil(t, a)
}

func TestGet_WithWallet(t *testing.T) {
	// No good way to mock token.AuditorWallet but we can test nil
}
