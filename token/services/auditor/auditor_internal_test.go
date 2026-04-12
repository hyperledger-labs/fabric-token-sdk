/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package auditor — internal tests for metrics noop types and requestWrapper.
// These tests remain in package auditor because they access unexported types.
package auditor

import (
	"context"
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	commondrivermock "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	drivermock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	tokenmock "github.com/hyperledger-labs/fabric-token-sdk/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared test helpers used across test files in this package.
// ---------------------------------------------------------------------------

// minimalRequest builds a minimal token.Request suitable for requestWrapper tests.
func minimalRequest(anchor string) *token.Request {
	return &token.Request{
		Anchor:   token.RequestAnchor(anchor),
		Actions:  &driver.TokenRequest{},
		Metadata: &driver.TokenRequestMetadata{},
	}
}

// ---------------------------------------------------------------------------
// newMetrics / Provider tests
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
	mp := &commondrivermock.MetricsProvider{}
	mp.NewCounterReturns(&noopCounter{})
	mp.NewGaugeReturns(&noopGauge{})
	mp.NewHistogramReturns(&noopHistogram{})

	m := newMetrics(mp)
	require.NotNil(t, m)
	// AuditLockConflicts, AppendErrors, ReleasesTotal = 3 counters
	assert.Equal(t, 3, mp.NewCounterCallCount())
	// AuditDuration, AppendDuration = 2 histograms
	assert.Equal(t, 2, mp.NewHistogramCallCount())
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
// requestWrapper tests
// ---------------------------------------------------------------------------

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
// Metrics integration tests (uses unexported noopProvider types)
// ---------------------------------------------------------------------------

func TestMetricsProviderCall(t *testing.T) {
	m := newMetrics(&noopProvider{})

	assert.NotPanics(t, func() {
		m.AuditLockConflicts.Add(1)
		m.AppendErrors.Add(1)
		m.ReleasesTotal.Add(1)

		m.AuditDuration.Observe(1.0)
		m.AppendDuration.Observe(1.0)
	})

	nc := &noopCounter{}
	assert.NotPanics(t, func() {
		nc.Add(12)
	})

	ng := &noopGauge{}
	assert.NotPanics(t, func() {
		ng.Add(12)
		ng.Set(12)
	})

	nh := &noopHistogram{}
	assert.NotPanics(t, func() {
		nh.Observe(12)
	})
}

// ---------------------------------------------------------------------------
// requestWrapper tests — access unexported types directly within package auditor
// ---------------------------------------------------------------------------

func newInternalTestManagementService(t *testing.T) *token.ManagementService {
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

func newInternalTestManagementServiceWithTokens(t *testing.T, toks []*token2.Token) *token.ManagementService {
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
	mockQE.ListAuditTokensReturns(toks, nil)
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

func TestRequestWrapper_PublicParamsHash(t *testing.T) {
	rw := newRequestWrapper(minimalRequest("tx-pph"), nil)
	assert.Panics(t, func() {
		rw.PublicParamsHash()
	})
}

func TestRequestWrapper_CompleteInputsWithEmptyEID_Shortcut(t *testing.T) {
	tms := newInternalTestManagementService(t)
	rw := newRequestWrapper(
		token.NewRequest(tms, token.RequestAnchor("tx-cid")), tms,
	)
	record := &token.AuditRecord{
		Inputs: token.NewInputStream(nil, []*token.Input{}, 0),
	}
	err := rw.completeInputsWithEmptyEID(context.Background(), record)
	assert.NoError(t, err)
}

func TestRequestWrapper_CompleteInputsWithEmptyEID_WithInputs(t *testing.T) {
	tmsWithToken := newInternalTestManagementServiceWithTokens(t, []*token2.Token{
		{Type: "USD", Quantity: "100", Owner: []byte("owner1")},
	})
	rw := newRequestWrapper(
		token.NewRequest(tmsWithToken, token.RequestAnchor("tx-cid2")), tmsWithToken,
	)
	recordWithInputs := &token.AuditRecord{
		Inputs:  token.NewInputStream(nil, []*token.Input{{Id: &token2.ID{TxId: "123"}}}, 0),
		Outputs: token.NewOutputStream([]*token.Output{{EnrollmentID: "target"}}, 0),
	}
	err := rw.completeInputsWithEmptyEID(context.Background(), recordWithInputs)
	assert.NoError(t, err)
}

func TestRequestWrapper_AuditRecord(t *testing.T) {
	tms := newInternalTestManagementService(t)
	rw := newRequestWrapper(
		token.NewRequest(tms, token.RequestAnchor("tx-ar")), tms,
	)
	record, err := rw.AuditRecord(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, record)
}

func TestRequestWrapper_AuditRecord_RequestError(t *testing.T) {
	// nil PublicParameters forces r.r.AuditRecord to return an error.
	mockTMS := &drivermock.TokenManagerService{}
	mockVP := &tokenmock.VaultProvider{}
	mockPPM := &drivermock.PublicParamsManager{}
	mockPPM.PublicParametersReturns(nil)
	mockTMS.PublicParamsManagerReturns(mockPPM)
	mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)
	mockTMS.TokensServiceReturns(&drivermock.TokensService{})
	mockTMS.WalletServiceReturns(&drivermock.WalletService{})
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(&drivermock.QueryEngine{})
	mockVP.VaultReturns(mockV, nil)

	badTMS, err := token.NewManagementService(
		token.TMSID{}, mockTMS, logging.MustGetLogger("test"), mockVP, nil, nil,
	)
	require.NoError(t, err)

	rw := newRequestWrapper(token.NewRequest(badTMS, token.RequestAnchor("tx-aud-rec-err")), badTMS)
	_, err = rw.AuditRecord(context.Background())
	require.Error(t, err)
}

func TestCompleteInputsWithEmptyEID_ListTokensError(t *testing.T) {
	mockTMS := &drivermock.TokenManagerService{}
	mockVP := &tokenmock.VaultProvider{}
	mockPPM := &drivermock.PublicParamsManager{}
	mockPP := &drivermock.PublicParameters{}
	mockPP.PrecisionReturns(64)
	mockPPM.PublicParametersReturns(mockPP)
	mockTMS.PublicParamsManagerReturns(mockPPM)
	mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)
	mockTMS.TokensServiceReturns(&drivermock.TokensService{})
	mockTMS.WalletServiceReturns(&drivermock.WalletService{})
	mockTMS.IssueServiceReturns(&drivermock.IssueService{})
	mockTMS.TransferServiceReturns(&drivermock.TransferService{})

	mockQE := &drivermock.QueryEngine{}
	mockQE.ListAuditTokensReturns(nil, errors.New("list tokens error"))
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(mockQE)
	mockVP.VaultReturns(mockV, nil)

	tms, err := token.NewManagementService(
		token.TMSID{}, mockTMS, logging.MustGetLogger("test"), mockVP, nil, nil,
	)
	require.NoError(t, err)

	rw := newRequestWrapper(token.NewRequest(tms, token.RequestAnchor("tx-list-err")), tms)
	record := &token.AuditRecord{
		Inputs:  token.NewInputStream(nil, []*token.Input{{Id: &token2.ID{TxId: "123"}}}, 0),
		Outputs: token.NewOutputStream([]*token.Output{{EnrollmentID: "target"}}, 0),
	}
	err = rw.completeInputsWithEmptyEID(context.Background(), record)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed listing tokens")
}

func TestCompleteInputsWithEmptyEID_ToQuantityError(t *testing.T) {
	mockTMS := &drivermock.TokenManagerService{}
	mockVP := &tokenmock.VaultProvider{}
	mockPPM := &drivermock.PublicParamsManager{}
	mockPP := &drivermock.PublicParameters{}
	mockPP.PrecisionReturns(64)
	mockPPM.PublicParametersReturns(mockPP)
	mockTMS.PublicParamsManagerReturns(mockPPM)
	mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)
	mockTMS.TokensServiceReturns(&drivermock.TokensService{})
	mockTMS.WalletServiceReturns(&drivermock.WalletService{})
	mockTMS.IssueServiceReturns(&drivermock.IssueService{})
	mockTMS.TransferServiceReturns(&drivermock.TransferService{})

	mockQE := &drivermock.QueryEngine{}
	mockQE.ListAuditTokensReturns([]*token2.Token{
		{Type: "USD", Quantity: "NOT_A_VALID_QUANTITY", Owner: []byte("owner1")},
	}, nil)
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(mockQE)
	mockVP.VaultReturns(mockV, nil)

	tms, err := token.NewManagementService(
		token.TMSID{}, mockTMS, logging.MustGetLogger("test"), mockVP, nil, nil,
	)
	require.NoError(t, err)

	rw := newRequestWrapper(token.NewRequest(tms, token.RequestAnchor("tx-qty-err")), tms)
	record := &token.AuditRecord{
		Inputs:  token.NewInputStream(nil, []*token.Input{{Id: &token2.ID{TxId: "123"}}}, 0),
		Outputs: token.NewOutputStream([]*token.Output{{EnrollmentID: "target"}}, 0),
	}
	err = rw.completeInputsWithEmptyEID(context.Background(), record)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed converting token quantity")
}
