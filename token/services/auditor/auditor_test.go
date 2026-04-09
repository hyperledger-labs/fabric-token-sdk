/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"context"
	"errors"
	"io"
	"testing"

	cdriver "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	drivermock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	tokenmock "github.com/hyperledger-labs/fabric-token-sdk/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers / stubs

func newTestManagementService(t *testing.T) *token.ManagementService {
	mockTMS := &drivermock.TokenManagerService{}
	mockVP := &tokenmock.VaultProvider{}

	mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)

	mockPPM := &drivermock.PublicParamsManager{}
	mockPP := &drivermock.PublicParameters{}
	mockPP.PrecisionReturns(64)
	mockPPM.PublicParametersReturns(mockPP)

	mockTMS.PublicParamsManagerReturns(mockPPM)

	// Provide TokensService, WalletService, IssueService, TransferService
	// so that GetMetadata() and inputsAndOutputs() work without panics.
	mockTMS.TokensServiceReturns(&drivermock.TokensService{})
	mockTMS.WalletServiceReturns(&drivermock.WalletService{})
	mockTMS.IssueServiceReturns(&drivermock.IssueService{})
	mockTMS.TransferServiceReturns(&drivermock.TransferService{})

	mockQE := &drivermock.QueryEngine{}
	// Return empty tokens by default (matches empty inputs from a request with no actions).
	mockQE.ListAuditTokensReturns([]*token2.Token{}, nil)
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(mockQE)
	mockVP.VaultReturns(mockV, nil)

	logger := logging.MustGetLogger("test")

	tms, err := token.NewManagementService(
		token.TMSID{},
		mockTMS,
		logger,
		mockVP,
		nil, // CertificationClientProvider
		nil, // SelectorManagerProvider
	)
	require.NoError(t, err)
	require.NotNil(t, tms)
	return tms
}

// newTestManagementServiceWithTokens is like newTestManagementService but
// configures ListAuditTokens to return the specified tokens.
func newTestManagementServiceWithTokens(t *testing.T, tokens []*token2.Token) *token.ManagementService {
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
	mockQE.ListAuditTokensReturns(tokens, nil)
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(mockQE)
	mockVP.VaultReturns(mockV, nil)

	logger := logging.MustGetLogger("test")
	tms, err := token.NewManagementService(
		token.TMSID{},
		mockTMS,
		logger,
		mockVP,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, tms)
	return tms
}
// ---------------------------------------------------------------------------

// stubMetricsProvider records how many times each factory method is called.
type stubMetricsProvider struct {
	counterCalls   int
	histogramCalls int
}

func (s *stubMetricsProvider) NewCounter(_ metrics.CounterOpts) metrics.Counter {
	s.counterCalls++
	return &noopCounter{}
}
func (s *stubMetricsProvider) NewGauge(_ metrics.GaugeOpts) metrics.Gauge { return &noopGauge{} }
func (s *stubMetricsProvider) NewHistogram(_ metrics.HistogramOpts) metrics.Histogram {
	s.histogramCalls++
	return &noopHistogram{}
}

// mockCheckService is an in-package mock for the CheckService interface.
type mockCheckService struct {
	issues []string
	err    error
}

func (m *mockCheckService) Check(_ context.Context) ([]string, error) {
	return m.issues, m.err
}

// mockServiceProvider implements token.ServiceProvider for GetByTMSID tests.
type mockServiceProvider struct {
	service interface{}
	err     error
}

func (m *mockServiceProvider) GetService(_ interface{}) (interface{}, error) {
	return m.service, m.err
}

type stubAuditTransactionStore struct {
	getStatusResult dbdriver.TxStatus
	getStatusMsg    string
	getStatusErr    error
	getTokenReqData []byte
	getTokenReqErr  error
	setStatusErr    error
	appendErr       error
}

func (s *stubAuditTransactionStore) Close() error { return nil }
func (s *stubAuditTransactionStore) BeginAtomicWrite() (dbdriver.AtomicWrite, error) {
	return &stubAtomicWrite{err: s.appendErr}, nil
}
func (s *stubAuditTransactionStore) SetStatus(_ context.Context, _ string, _ dbdriver.TxStatus, _ string) error {
	return s.setStatusErr
}
func (s *stubAuditTransactionStore) GetStatus(_ context.Context, _ string) (dbdriver.TxStatus, string, error) {
	return s.getStatusResult, s.getStatusMsg, s.getStatusErr
}
func (s *stubAuditTransactionStore) QueryTransactions(_ context.Context, _ dbdriver.QueryTransactionsParams, _ cdriver.Pagination) (*cdriver.PageIterator[*dbdriver.TransactionRecord], error) {
	return nil, nil
}
func (s *stubAuditTransactionStore) QueryMovements(_ context.Context, _ dbdriver.QueryMovementsParams) ([]*dbdriver.MovementRecord, error) {
	return nil, nil
}
func (s *stubAuditTransactionStore) QueryValidations(_ context.Context, _ dbdriver.QueryValidationRecordsParams) (dbdriver.ValidationRecordsIterator, error) {
	return nil, nil
}
func (s *stubAuditTransactionStore) QueryTokenRequests(_ context.Context, _ dbdriver.QueryTokenRequestsParams) (dbdriver.TokenRequestIterator, error) {
	return &stubTokenRequestIterator{count: 1}, nil
}

type stubTokenRequestIterator struct {
	count int
}

func (s *stubTokenRequestIterator) Next() (*dbdriver.TokenRequestRecord, error) {
	if s.count > 0 {
		s.count--
		return &dbdriver.TokenRequestRecord{TxID: "txid-123"}, nil
	}
	// Return io.EOF
	return nil, io.EOF
}

func (s *stubTokenRequestIterator) Close() {}

func (s *stubAuditTransactionStore) GetTokenRequest(_ context.Context, _ string) ([]byte, error) {
	return s.getTokenReqData, s.getTokenReqErr
}

// stubAtomicWrite is a no-op AtomicWrite (used in stubAuditTransactionStore.BeginAtomicWrite).
type stubAtomicWrite struct{
	err error
}

func (a *stubAtomicWrite) Commit() error { return a.err }
func (a *stubAtomicWrite) Rollback()     {}
func (a *stubAtomicWrite) AddTokenRequest(_ context.Context, _ string, _ []byte, _, _ map[string][]byte, _ driver.PPHash) error {
	return nil
}
func (a *stubAtomicWrite) AddMovement(_ context.Context, _ ...dbdriver.MovementRecord) error {
	return nil
}
func (a *stubAtomicWrite) AddTransaction(_ context.Context, _ ...dbdriver.TransactionRecord) error {
	return nil
}
func (a *stubAtomicWrite) AddValidationRecord(_ context.Context, _ string, _ map[string][]byte) error {
	return nil
}

// newTestStoreService builds a *auditdb.StoreService backed by the given stub.
func newTestStoreService(t *testing.T, stub dbdriver.AuditTransactionStore) *auditdb.StoreService {
	t.Helper()
	ss, err := auditdb.NewStoreServiceForTest(stub)
	require.NoError(t, err)
	return ss
}

// mockTransaction is a minimal Transaction for Release tests.
type mockTransaction struct {
	anchor string
	tms    *token.ManagementService
}

func (m *mockTransaction) ID() string            { return m.anchor }
func (m *mockTransaction) Network() string        { return "testnet" }
func (m *mockTransaction) Channel() string        { return "testch" }
func (m *mockTransaction) Namespace() string      { return "testns" }
func (m *mockTransaction) Request() *token.Request {
	if m.tms != nil {
		return token.NewRequest(m.tms, token.RequestAnchor(m.anchor))
	}
	return &token.Request{Anchor: token.RequestAnchor(m.anchor)}
}

// ---------------------------------------------------------------------------
// metrics.go tests
// ---------------------------------------------------------------------------

func TestNewMetrics_NilProvider(t *testing.T) {
	m := newMetrics(nil)
	require.NotNil(t, m)
	assert.NotNil(t, m.AuditDuration)
	assert.NotNil(t, m.AuditLockConflicts)
	assert.NotNil(t, m.AppendDuration)
	assert.NotNil(t, m.AppendErrors)
	assert.NotNil(t, m.ReleasesTotal)
}

func TestNewMetrics_WithProvider(t *testing.T) {
	p := &stubMetricsProvider{}
	m := newMetrics(p)
	require.NotNil(t, m)
	// AuditLockConflicts, AppendErrors, ReleasesTotal = 3 counters
	assert.Equal(t, 3, p.counterCalls)
	// AuditDuration, AppendDuration = 2 histograms
	assert.Equal(t, 2, p.histogramCalls)
}

func TestNoopCounter_With_ReturnsSelf(t *testing.T) {
	c := &noopCounter{}
	c2 := c.With("key", "val")
	assert.Equal(t, c, c2)
}

func TestNoopCounter_Add_NoPanic(t *testing.T) {
	c := &noopCounter{}
	assert.NotPanics(t, func() { c.Add(3.14) })
}

func TestNoopGauge_With_ReturnsSelf(t *testing.T) {
	g := &noopGauge{}
	g2 := g.With("key", "val")
	assert.Equal(t, g, g2)
}

func TestNoopGauge_Add_NoPanic(t *testing.T) {
	g := &noopGauge{}
	assert.NotPanics(t, func() { g.Add(1.5) })
}

func TestNoopGauge_Set_NoPanic(t *testing.T) {
	g := &noopGauge{}
	assert.NotPanics(t, func() { g.Set(42.0) })
}

func TestNoopHistogram_With_ReturnsSelf(t *testing.T) {
	h := &noopHistogram{}
	h2 := h.With("key", "val")
	assert.Equal(t, h, h2)
}

func TestNoopHistogram_Observe_NoPanic(t *testing.T) {
	h := &noopHistogram{}
	assert.NotPanics(t, func() { h.Observe(0.001) })
}

func TestNoopProvider_NewCounter_ReturnsNoopCounter(t *testing.T) {
	p := &noopProvider{}
	c := p.NewCounter(metrics.CounterOpts{Name: "x"})
	require.NotNil(t, c)
	_, ok := c.(*noopCounter)
	assert.True(t, ok)
}

func TestNoopProvider_NewGauge_ReturnsNoopGauge(t *testing.T) {
	p := &noopProvider{}
	g := p.NewGauge(metrics.GaugeOpts{Name: "y"})
	require.NotNil(t, g)
	_, ok := g.(*noopGauge)
	assert.True(t, ok)
}

func TestNoopProvider_NewHistogram_ReturnsNoopHistogram(t *testing.T) {
	p := &noopProvider{}
	h := p.NewHistogram(metrics.HistogramOpts{Name: "z", Buckets: []float64{1}})
	require.NotNil(t, h)
	_, ok := h.(*noopHistogram)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// requestWrapper tests (auditor.go)
// ---------------------------------------------------------------------------

func minimalRequest(anchor string) *token.Request {
	return &token.Request{
		Anchor:   token.RequestAnchor(anchor),
		Actions:  &driver.TokenRequest{},
		Metadata: &driver.TokenRequestMetadata{},
	}
}

func TestRequestWrapper_ID(t *testing.T) {
	rw := newRequestWrapper(minimalRequest("tx-001"), nil)
	assert.Equal(t, token.RequestAnchor("tx-001"), rw.ID())
}

func TestRequestWrapper_String(t *testing.T) {
	rw := newRequestWrapper(minimalRequest("tx-hello"), nil)
	assert.Equal(t, "tx-hello", rw.String())
}

func TestRequestWrapper_Bytes_ValidRequest(t *testing.T) {
	rw := newRequestWrapper(minimalRequest("tx-002"), nil)
	b, err := rw.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestRequestWrapper_AllApplicationMetadata_Nil(t *testing.T) {
	req := &token.Request{
		Anchor:   "tx-003",
		Metadata: &driver.TokenRequestMetadata{Application: nil},
	}
	rw := newRequestWrapper(req, nil)
	assert.Nil(t, rw.AllApplicationMetadata())
}

func TestRequestWrapper_AllApplicationMetadata_Populated(t *testing.T) {
	req := &token.Request{
		Anchor: "tx-004",
		Metadata: &driver.TokenRequestMetadata{
			Application: map[string][]byte{"k": []byte("v")},
		},
	}
	rw := newRequestWrapper(req, nil)
	m := rw.AllApplicationMetadata()
	require.NotNil(t, m)
	assert.Equal(t, []byte("v"), m["k"])
}

// ---------------------------------------------------------------------------
// Service.Check tests (auditor.go)
// ---------------------------------------------------------------------------

func newServiceWithCheckService(cs CheckService) *Service {
	return &Service{
		metrics:      newMetrics(nil),
		checkService: cs,
	}
}

func TestService_Check_ReturnsIssues(t *testing.T) {
	svc := newServiceWithCheckService(&mockCheckService{issues: []string{"tx-aaa", "tx-bbb"}})
	got, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"tx-aaa", "tx-bbb"}, got)
}

func TestService_Check_ReturnsError(t *testing.T) {
	expectedErr := errors.New("check failed")
	svc := newServiceWithCheckService(&mockCheckService{err: expectedErr})
	_, err := svc.Check(context.Background())
	assert.ErrorIs(t, err, expectedErr)
}

func TestService_Check_EmptyIssues(t *testing.T) {
	svc := newServiceWithCheckService(&mockCheckService{issues: []string{}})
	got, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)
}

// ---------------------------------------------------------------------------
// manager.go — Get / GetByTMSID
// ---------------------------------------------------------------------------

func TestGet_NilWallet_ReturnsNil(t *testing.T) {
	// w == nil → early return, sp is never accessed
	got := Get(nil, nil)
	assert.Nil(t, got)
}

func TestGetByTMSID_GetServiceError_ReturnsNil(t *testing.T) {
	sp := &mockServiceProvider{err: errors.New("registry lookup failed")}
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	got := GetByTMSID(sp, tmsID)
	assert.Nil(t, got)
}

// ---------------------------------------------------------------------------
// Service.Release / SetStatus / GetStatus / GetTokenRequest
// ---------------------------------------------------------------------------

func newServiceWithAuditDB(t *testing.T, stub dbdriver.AuditTransactionStore) *Service {
	t.Helper()
	return &Service{
		metrics: newMetrics(nil),
		auditDB: newTestStoreService(t, stub),
	}
}

func TestService_Release_IncrementsCounter(t *testing.T) {
	// counters are noops; we verify Release doesn't panic and completes cleanly
	svc := newServiceWithAuditDB(t, &stubAuditTransactionStore{})
	tx := &mockTransaction{anchor: "tx-release"}
	assert.NotPanics(t, func() {
		svc.Release(context.Background(), tx)
	})
}

func TestService_SetStatus_Success(t *testing.T) {
	svc := newServiceWithAuditDB(t, &stubAuditTransactionStore{})
	err := svc.SetStatus(context.Background(), "tx-set", auditdb.Confirmed, "ok")
	assert.NoError(t, err)
}

func TestService_SetStatus_Error(t *testing.T) {
	expectedErr := errors.New("db write error")
	svc := newServiceWithAuditDB(t, &stubAuditTransactionStore{setStatusErr: expectedErr})
	err := svc.SetStatus(context.Background(), "tx-set", auditdb.Confirmed, "ok")
	assert.ErrorIs(t, err, expectedErr)
}

func TestService_GetStatus_Success(t *testing.T) {
	stub := &stubAuditTransactionStore{
		getStatusResult: auditdb.Confirmed,
		getStatusMsg:    "done",
	}
	svc := newServiceWithAuditDB(t, stub)
	status, msg, err := svc.GetStatus(context.Background(), "tx-get")
	require.NoError(t, err)
	assert.Equal(t, auditdb.Confirmed, status)
	assert.Equal(t, "done", msg)
}

func TestService_GetStatus_Error(t *testing.T) {
	expectedErr := errors.New("db read error")
	svc := newServiceWithAuditDB(t, &stubAuditTransactionStore{getStatusErr: expectedErr})
	_, _, err := svc.GetStatus(context.Background(), "tx-get")
	assert.ErrorIs(t, err, expectedErr)
}

func TestService_GetTokenRequest_Success(t *testing.T) {
	data := []byte("raw-token-request")
	svc := newServiceWithAuditDB(t, &stubAuditTransactionStore{getTokenReqData: data})
	got, err := svc.GetTokenRequest(context.Background(), "tx-tok")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestService_GetTokenRequest_Error(t *testing.T) {
	expectedErr := errors.New("not found")
	svc := newServiceWithAuditDB(t, &stubAuditTransactionStore{getTokenReqErr: expectedErr})
	_, err := svc.GetTokenRequest(context.Background(), "tx-tok")
	assert.ErrorIs(t, err, expectedErr)
}
