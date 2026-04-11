/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package auditor — internal tests for metrics noop types and requestWrapper.
// These tests remain in package auditor because they access unexported types.
package auditor

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	commondrivermock "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared test helpers (used across internal and external test files via
// export_test.go wrappers).
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
