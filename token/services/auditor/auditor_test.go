/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	drivermock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	tokenmock "github.com/hyperledger-labs/fabric-token-sdk/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	auditmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	auditdbmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb/mock"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	depmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

// fakeServiceProvider is a simple test stub implementing token.ServiceProvider.
type fakeServiceProvider struct {
	service any
	err     error
}

func (f *fakeServiceProvider) GetService(_ any) (any, error) {
	return f.service, f.err
}

// ---------------------------------------------------------------------------
// Shared test helpers
// ---------------------------------------------------------------------------

func newTestManagementService(t *testing.T) *token.ManagementService {
	t.Helper()
	mockTMS := &drivermock.TokenManagerService{}
	mockVP := &tokenmock.VaultProvider{}

	mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)

	mockPPM := &drivermock.PublicParamsManager{}
	mockPP := &drivermock.PublicParameters{}
	mockPP.PrecisionReturns(64)
	mockPPM.PublicParametersReturns(mockPP)

	mockTMS.PublicParamsManagerReturns(mockPPM)
	mockTMS.TokensServiceReturns(&drivermock.TokensService{})
	mockTMS.WalletServiceReturns(&drivermock.WalletService{})
	mockTMS.IssueServiceReturns(&drivermock.IssueService{})
	mockTMS.TransferServiceReturns(&drivermock.TransferService{})

	mockQE := &drivermock.QueryEngine{}
	mockQE.ListAuditTokensReturns([]*token2.Token{}, nil)
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(mockQE)
	mockVP.VaultReturns(mockV, nil)

	tms, err := token.NewManagementService(
		token.TMSID{},
		mockTMS,
		logging.MustGetLogger("test"),
		mockVP,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, tms)

	return tms
}

// stubTokenRequestIterator is a minimal test helper for returning token request records.
type stubTokenRequestIterator struct {
	count int
}

func (s *stubTokenRequestIterator) Next() (*dbdriver.TokenRequestRecord, error) {
	if s.count > 0 {
		s.count--

		return &dbdriver.TokenRequestRecord{TxID: "txid-123"}, nil
	}

	return nil, io.EOF
}

func (s *stubTokenRequestIterator) Close() {}

// newTestStoreService builds a *auditdb.StoreService backed by the given store mock.
func newTestStoreService(t *testing.T, store dbdriver.AuditTransactionStore) *auditdb.StoreService {
	t.Helper()
	ss, err := auditdb.NewStoreService(store)
	require.NoError(t, err)

	return ss
}

// newFakeStore returns a counterfeiter AuditTransactionStore with default no-op behaviour.
func newFakeStore() *auditmock.AuditTransactionStore {
	fakeStore := &auditmock.AuditTransactionStore{}
	fakeAtomicWrite := &auditmock.AtomicWrite{}
	fakeStore.BeginAtomicWriteReturns(fakeAtomicWrite, nil)
	fakeStore.QueryTokenRequestsStub = func(_ context.Context, _ dbdriver.QueryTokenRequestsParams) (dbdriver.TokenRequestIterator, error) {
		return &stubTokenRequestIterator{count: 1}, nil
	}

	return fakeStore
}

// newTestService creates a Service with the given auditDB and checkService for testing.
func newTestService(auditDB *auditdb.StoreService, checkService auditor.CheckService) *auditor.Service {
	return auditor.NewService(
		token.TMSID{},
		nil, // networkProvider
		auditDB,
		nil, // tokenDB
		nil, // tmsProvider
		nil, // finalityTracer
		nil, // metricsProvider
		checkService,
	)
}

// newStubNetwork creates a *network.Network backed by a no-op counterfeiter Network fake.
func newStubNetwork() *network.Network {
	return network.NewNetwork(&auditmock.Network{}, nil)
}

// ---------------------------------------------------------------------------
// Service.Check tests
// ---------------------------------------------------------------------------

func TestService_Check_ReturnsIssues(t *testing.T) {
	cs := &auditmock.CheckService{}
	cs.CheckReturns([]string{"tx-aaa", "tx-bbb"}, nil)
	svc := newTestService(newTestStoreService(t, newFakeStore()), cs)
	got, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"tx-aaa", "tx-bbb"}, got)
}

func TestService_Check_ReturnsError(t *testing.T) {
	expectedErr := errors.New("check failed")
	cs := &auditmock.CheckService{}
	cs.CheckReturns(nil, expectedErr)
	svc := newTestService(newTestStoreService(t, newFakeStore()), cs)
	_, err := svc.Check(context.Background())
	assert.ErrorIs(t, err, expectedErr)
}

func TestService_Check_EmptyIssues(t *testing.T) {
	cs := &auditmock.CheckService{}
	cs.CheckReturns([]string{}, nil)
	svc := newTestService(newTestStoreService(t, newFakeStore()), cs)
	got, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)
}

// ---------------------------------------------------------------------------
// manager.go — Get / GetByTMSID
// ---------------------------------------------------------------------------

func TestGet_NilWallet_ReturnsNil(t *testing.T) {
	got := auditor.Get(nil, nil)
	assert.Nil(t, got)
}

func TestGetByTMSID_GetServiceError_ReturnsNil(t *testing.T) {
	sp := &fakeServiceProvider{err: errors.New("registry lookup failed")}
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	got := auditor.GetByTMSID(sp, tmsID)
	assert.Nil(t, got)
}

// ---------------------------------------------------------------------------
// Service.Release / SetStatus / GetStatus / GetTokenRequest
// ---------------------------------------------------------------------------

func TestService_Release_IncrementsCounter(t *testing.T) {
	svc := newTestService(newTestStoreService(t, newFakeStore()), nil)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-release")
	tx.RequestReturns(&token.Request{Anchor: "tx-release"})
	assert.NotPanics(t, func() {
		svc.Release(context.Background(), tx)
	})
}

func TestService_SetStatus_Success(t *testing.T) {
	svc := newTestService(newTestStoreService(t, newFakeStore()), nil)
	err := svc.SetStatus(context.Background(), "tx-set", auditdb.Confirmed, "ok")
	assert.NoError(t, err)
}

func TestService_SetStatus_Error(t *testing.T) {
	expectedErr := errors.New("db write error")
	fakeStore := newFakeStore()
	fakeStore.SetStatusReturns(expectedErr)
	svc := newTestService(newTestStoreService(t, fakeStore), nil)
	err := svc.SetStatus(context.Background(), "tx-set", auditdb.Confirmed, "ok")
	assert.ErrorIs(t, err, expectedErr)
}

func TestService_GetStatus_Success(t *testing.T) {
	fakeStore := newFakeStore()
	fakeStore.GetStatusReturns(auditdb.Confirmed, "done", nil)
	svc := newTestService(newTestStoreService(t, fakeStore), nil)
	status, msg, err := svc.GetStatus(context.Background(), "tx-get")
	require.NoError(t, err)
	assert.Equal(t, auditdb.Confirmed, status)
	assert.Equal(t, "done", msg)
}

func TestService_GetStatus_Error(t *testing.T) {
	expectedErr := errors.New("db read error")
	fakeStore := newFakeStore()
	fakeStore.GetStatusReturns(0, "", expectedErr)
	svc := newTestService(newTestStoreService(t, fakeStore), nil)
	_, _, err := svc.GetStatus(context.Background(), "tx-get")
	assert.ErrorIs(t, err, expectedErr)
}

func TestService_GetTokenRequest_Success(t *testing.T) {
	data := []byte("raw-token-request")
	fakeStore := newFakeStore()
	fakeStore.GetTokenRequestReturns(data, nil)
	svc := newTestService(newTestStoreService(t, fakeStore), nil)
	got, err := svc.GetTokenRequest(context.Background(), "tx-tok")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestService_GetTokenRequest_Error(t *testing.T) {
	expectedErr := errors.New("not found")
	fakeStore := newFakeStore()
	fakeStore.GetTokenRequestReturns(nil, expectedErr)
	svc := newTestService(newTestStoreService(t, fakeStore), nil)
	_, err := svc.GetTokenRequest(context.Background(), "tx-tok")
	assert.ErrorIs(t, err, expectedErr)
}

// ---------------------------------------------------------------------------
// Service.Validate tests
// ---------------------------------------------------------------------------

func TestService_Validate(t *testing.T) {
	svc := newTestService(nil, nil)
	assert.Panics(t, func() {
		_ = svc.Validate(context.Background(), &token.Request{})
	})
}

// ---------------------------------------------------------------------------
// Service.Audit tests
// ---------------------------------------------------------------------------

func TestService_Audit_AuditRecordError(t *testing.T) {
	mockTMS := &drivermock.TokenManagerService{}
	mockPPM := &drivermock.PublicParamsManager{}
	mockPPM.PublicParametersReturns(nil)
	mockTMS.PublicParamsManagerReturns(mockPPM)
	mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)
	mockTMS.TokensServiceReturns(&drivermock.TokensService{})
	mockTMS.WalletServiceReturns(&drivermock.WalletService{})

	mockVP := &tokenmock.VaultProvider{}
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(&drivermock.QueryEngine{})
	mockVP.VaultReturns(mockV, nil)

	badTMS, err := token.NewManagementService(
		token.TMSID{}, mockTMS, logging.MustGetLogger("test"), mockVP, nil, nil,
	)
	require.NoError(t, err)

	svc := newTestService(newTestStoreService(t, newFakeStore()), nil)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-err")
	tx.RequestReturns(token.NewRequest(badTMS, token.RequestAnchor("tx-err")))

	_, _, err = svc.Audit(context.Background(), tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed getting transaction audit record")
}

func TestService_Audit_Success(t *testing.T) {
	svc := newTestService(newTestStoreService(t, newFakeStore()), nil)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-audit-ok")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-audit-ok")))

	inputs, outputs, err := svc.Audit(context.Background(), tx)
	require.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}

func TestService_Audit_DBCleanSuccess(t *testing.T) {
	fakeStore := newFakeStore()
	fakeStore.GetStatusReturns(0, "", errors.New("db status err"))

	svc := newTestService(newTestStoreService(t, fakeStore), nil)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-aud-err")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-aud-err")))

	inputs, outputs, err := svc.Audit(context.Background(), tx)
	require.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}

func TestService_Audit_NotUnknown(t *testing.T) {
	fakeStore := newFakeStore()
	fakeStore.GetStatusReturns(dbdriver.Pending, "", nil)

	svc := newTestService(newTestStoreService(t, fakeStore), nil)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-aud-not-unknown")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-aud-not-unknown")))

	inputs, outputs, err := svc.Audit(context.Background(), tx)
	require.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}

func TestService_Audit_TMSProviderIrrelevant(t *testing.T) {
	tmsProv := &depmock.TokenManagementServiceProvider{}
	tmsProv.TokenManagementServiceReturns(nil, errors.New("tms err"))

	svc := auditor.NewService(
		token.TMSID{}, nil,
		newTestStoreService(t, newFakeStore()),
		nil, tmsProv, nil, nil, nil,
	)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-aud-tms-err")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-aud-tms-err")))

	inputs, outputs, err := svc.Audit(context.Background(), tx)
	require.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}

// ---------------------------------------------------------------------------
// Service.Append tests
// ---------------------------------------------------------------------------

func TestService_Append_Error_TMSProvider(t *testing.T) {
	tmsProv := &depmock.TokenManagementServiceProvider{}
	tmsProv.TokenManagementServiceReturns(nil, errors.New("tms err"))

	svc := auditor.NewService(
		token.TMSID{}, nil,
		newTestStoreService(t, newFakeStore()),
		nil, tmsProv, nil, nil, nil,
	)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-app")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-app")))

	err := svc.Append(context.Background(), tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tms err")
}

func TestService_Append_GetNetworkError(t *testing.T) {
	netProvider := &auditmock.NetworkProvider{}
	netProvider.GetNetworkReturns(nil, errors.New("network unavailable"))

	svc := auditor.NewService(
		token.TMSID{}, netProvider,
		newTestStoreService(t, newFakeStore()),
		nil, &depmock.TokenManagementServiceProvider{}, nil, nil, nil,
	)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-net-err")
	tx.NetworkReturns("testnet")
	tx.ChannelReturns("testch")
	tx.NamespaceReturns("testns")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-net-err")))

	err := svc.Append(context.Background(), tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed getting network instance")
}

func TestService_Append_Success(t *testing.T) {
	netProvider := &auditmock.NetworkProvider{}
	netProvider.GetNetworkReturns(newStubNetwork(), nil)

	svc := auditor.NewService(
		token.TMSID{}, netProvider,
		newTestStoreService(t, newFakeStore()),
		nil, &depmock.TokenManagementServiceProvider{}, nil, nil, nil,
	)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-app-success")
	tx.NetworkReturns("testnet")
	tx.ChannelReturns("testch")
	tx.NamespaceReturns("testns")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-app-success")))

	err := svc.Append(context.Background(), tx)
	require.NoError(t, err)
}

func TestService_Append_AddFinalityListenerError(t *testing.T) {
	fakeNet := &auditmock.Network{}
	fakeNet.AddFinalityListenerReturns(errors.New("listener fail"))

	netProvider := &auditmock.NetworkProvider{}
	netProvider.GetNetworkReturns(network.NewNetwork(fakeNet, nil), nil)

	svc := auditor.NewService(
		token.TMSID{}, netProvider,
		newTestStoreService(t, newFakeStore()),
		nil, &depmock.TokenManagementServiceProvider{}, nil, nil, nil,
	)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-listener-err")
	tx.NetworkReturns("testnet")
	tx.ChannelReturns("testch")
	tx.NamespaceReturns("testns")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-listener-err")))

	err := svc.Append(context.Background(), tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed listening to network")
}

func TestService_Append_AuditError(t *testing.T) {
	fakeStore := newFakeStore()
	fakeStore.BeginAtomicWriteStub = func() (dbdriver.AtomicWrite, error) {
		fakeAW := &auditmock.AtomicWrite{}
		fakeAW.CommitReturns(errors.New("db append err"))

		return fakeAW, nil
	}

	netProvider := &auditmock.NetworkProvider{}
	netProvider.GetNetworkReturns(newStubNetwork(), nil)

	svc := auditor.NewService(
		token.TMSID{}, netProvider,
		newTestStoreService(t, fakeStore),
		nil, &depmock.TokenManagementServiceProvider{}, nil, nil, nil,
	)
	tx := &auditmock.Transaction{}
	tx.IDReturns("tx-app-err")
	tx.NetworkReturns("testnet")
	tx.ChannelReturns("testch")
	tx.NamespaceReturns("testns")
	tx.RequestReturns(token.NewRequest(newTestManagementService(t), token.RequestAnchor("tx-app-err")))

	err := svc.Append(context.Background(), tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed appending request")
}

// ---------------------------------------------------------------------------
// ServiceManager tests
// ---------------------------------------------------------------------------

func TestNewServiceManager(t *testing.T) {
	sm := auditor.NewServiceManager(
		&auditmock.NetworkProvider{},
		&auditdbmock.AuditStoreServiceManager{},
		&auditmock.TokensServiceManager{},
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	assert.NotNil(t, sm)
}

func TestServiceManager_Auditor(t *testing.T) {
	netProv := &auditmock.NetworkProvider{}
	netProv.GetNetworkReturns(nil, errors.New("net err"))

	ssm := &auditdbmock.AuditStoreServiceManager{}
	ssm.StoreServiceByTMSIdReturns(nil, errors.New("db err"))

	tsm := &auditmock.TokensServiceManager{}
	tsm.ServiceByTMSIdReturns(nil, errors.New("tok err"))

	sm := auditor.NewServiceManager(
		netProv, ssm, tsm,
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	a, err := sm.Auditor(token.TMSID{Network: "n1", Channel: "c1", Namespace: "ns1"})
	require.Error(t, err)
	assert.Nil(t, a)
}

func TestServiceManager_RestoreTMS(t *testing.T) {
	// 1. Network error
	netProv := &auditmock.NetworkProvider{}
	netProv.GetNetworkReturns(nil, errors.New("net err"))

	sm := auditor.NewServiceManager(
		netProv,
		&auditdbmock.AuditStoreServiceManager{},
		&auditmock.TokensServiceManager{},
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	err := sm.RestoreTMS(token.TMSID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get network instance")

	// 2. Token service error (GetNetwork succeeds, ServiceByTMSId fails)
	tsm := &auditmock.TokensServiceManager{}
	tsm.ServiceByTMSIdReturns(nil, errors.New("tok err"))

	sm2 := auditor.NewServiceManager(
		&auditmock.NetworkProvider{},
		&auditdbmock.AuditStoreServiceManager{},
		tsm,
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	err = sm2.RestoreTMS(token.TMSID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get auditdb")
}

func TestServiceManager_RestoreTMS_GetAuditorError(t *testing.T) {
	// GetNetwork and ServiceByTMSId succeed at the outer level, but the lazy
	// factory fails because CheckService returns an error — so p.Get returns an error.
	netProv := &auditmock.NetworkProvider{}
	netProv.GetNetworkReturns(&network.Network{}, nil)

	ssm := &auditdbmock.AuditStoreServiceManager{}
	ssm.StoreServiceByTMSIdReturns(newTestStoreService(t, newFakeStore()), nil)

	tsm := &auditmock.TokensServiceManager{}
	tsm.ServiceByTMSIdReturns(&tokens.Service{}, nil)

	csp := &auditmock.CheckServiceProvider{}
	csp.CheckServiceReturns(nil, errors.New("checkservice err"))

	sm := auditor.NewServiceManager(
		netProv, ssm, tsm,
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		csp,
	)
	err := sm.RestoreTMS(token.TMSID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get auditor for")
}

func TestServiceManager_RestoreTMS_TokenRequestsError(t *testing.T) {
	// p.Get succeeds but the underlying store returns an error from QueryTokenRequests.
	netProv := &auditmock.NetworkProvider{}
	netProv.GetNetworkReturns(&network.Network{}, nil)

	errStore := newFakeStore()
	errStore.QueryTokenRequestsStub = func(_ context.Context, _ dbdriver.QueryTokenRequestsParams) (dbdriver.TokenRequestIterator, error) {
		return nil, errors.New("token requests err")
	}

	ssm := &auditdbmock.AuditStoreServiceManager{}
	ssm.StoreServiceByTMSIdReturns(newTestStoreService(t, errStore), nil)

	tsm := &auditmock.TokensServiceManager{}
	tsm.ServiceByTMSIdReturns(&tokens.Service{}, nil)

	sm := auditor.NewServiceManager(
		netProv, ssm, tsm,
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	err := sm.RestoreTMS(token.TMSID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get tx iterator")
}

func TestServiceManager_RestoreTMS_Success(t *testing.T) {
	netProv := &auditmock.NetworkProvider{}
	netProv.GetNetworkReturns(&network.Network{}, nil) // empty Network will panic in loop

	ssm := &auditdbmock.AuditStoreServiceManager{}
	ssm.StoreServiceByTMSIdReturns(newTestStoreService(t, newFakeStore()), nil)

	assert.Panics(t, func() {
		smSuccess := auditor.NewServiceManager(
			netProv,
			ssm,
			&auditmock.TokensServiceManager{},
			&depmock.TokenManagementServiceProvider{},
			noop.NewTracerProvider(),
			nil,
			&auditmock.CheckServiceProvider{},
		)
		_ = smSuccess.RestoreTMS(token.TMSID{})
	})
}

func TestServiceManager_Auditor_InitSuccess(t *testing.T) {
	ssm := &auditdbmock.AuditStoreServiceManager{}
	ssm.StoreServiceByTMSIdReturns(newTestStoreService(t, newFakeStore()), nil)

	tsm := &auditmock.TokensServiceManager{}
	tsm.ServiceByTMSIdReturns(&tokens.Service{}, nil)

	sm := auditor.NewServiceManager(
		&auditmock.NetworkProvider{},
		ssm, tsm,
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	a, err := sm.Auditor(token.TMSID{Network: "n1", Channel: "c1", Namespace: "ns1"})
	require.NoError(t, err)
	assert.NotNil(t, a)
}

// ---------------------------------------------------------------------------
// GetByTMSID closure error tests
// ---------------------------------------------------------------------------

func TestManager_GetByTMSID_ClosureErrors(t *testing.T) {
	sp := &fakeServiceProvider{}

	// 1. StoreServiceByTMSId error
	ssm := &auditdbmock.AuditStoreServiceManager{}
	ssm.StoreServiceByTMSIdReturns(nil, assert.AnError)

	smStoreErr := auditor.NewServiceManager(
		&auditmock.NetworkProvider{},
		ssm,
		&auditmock.TokensServiceManager{},
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	sp.service = smStoreErr
	assert.Nil(t, auditor.GetByTMSID(sp, token.TMSID{}))

	// 2. ServiceByTMSId error
	tsm := &auditmock.TokensServiceManager{}
	tsm.ServiceByTMSIdReturns(nil, assert.AnError)

	smTokensErr := auditor.NewServiceManager(
		&auditmock.NetworkProvider{},
		&auditdbmock.AuditStoreServiceManager{},
		tsm,
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	sp.service = smTokensErr
	assert.Nil(t, auditor.GetByTMSID(sp, token.TMSID{}))

	// 3. GetNetwork error
	netProv := &auditmock.NetworkProvider{}
	netProv.GetNetworkReturns(nil, assert.AnError)

	smNetworkErr := auditor.NewServiceManager(
		netProv,
		&auditdbmock.AuditStoreServiceManager{},
		&auditmock.TokensServiceManager{},
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	sp.service = smNetworkErr
	assert.Nil(t, auditor.GetByTMSID(sp, token.TMSID{}))

	// 4. CheckService error
	csp := &auditmock.CheckServiceProvider{}
	csp.CheckServiceReturns(nil, assert.AnError)

	smCheckErr := auditor.NewServiceManager(
		&auditmock.NetworkProvider{},
		&auditdbmock.AuditStoreServiceManager{},
		&auditmock.TokensServiceManager{},
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		csp,
	)
	sp.service = smCheckErr
	assert.Nil(t, auditor.GetByTMSID(sp, token.TMSID{}))
}

func TestManager_GetByTMSID(t *testing.T) {
	sp := &fakeServiceProvider{}

	// Error getting manager service
	sp.service = nil
	sp.err = assert.AnError
	a := auditor.GetByTMSID(sp, token.TMSID{})
	assert.Nil(t, a)

	// Success getting manager but Auditor returns error (network error)
	netProvErr := &auditmock.NetworkProvider{}
	netProvErr.GetNetworkReturns(nil, assert.AnError)

	sm := auditor.NewServiceManager(
		netProvErr,
		&auditdbmock.AuditStoreServiceManager{},
		&auditmock.TokensServiceManager{},
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	sp.service = sm
	sp.err = nil
	a = auditor.GetByTMSID(sp, token.TMSID{})
	assert.Nil(t, a)

	// Success Auditor
	ssm := &auditdbmock.AuditStoreServiceManager{}
	ssm.StoreServiceByTMSIdReturns(newTestStoreService(t, newFakeStore()), nil)

	tsm := &auditmock.TokensServiceManager{}
	tsm.ServiceByTMSIdReturns(&tokens.Service{}, nil)

	smSuccess := auditor.NewServiceManager(
		&auditmock.NetworkProvider{},
		ssm, tsm,
		&depmock.TokenManagementServiceProvider{},
		noop.NewTracerProvider(),
		nil,
		&auditmock.CheckServiceProvider{},
	)
	sp.service = smSuccess
	sp.err = nil
	a = auditor.GetByTMSID(sp, token.TMSID{})
	assert.NotNil(t, a)

	// Test Get: nil wallet
	a2 := auditor.Get(sp, nil)
	assert.Nil(t, a2)

	// non-nil wallet panics due to being empty
	w := &token.AuditorWallet{}
	assert.Panics(t, func() {
		auditor.Get(sp, w)
	})
}
