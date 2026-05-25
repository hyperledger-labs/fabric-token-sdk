/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity_test

import (
	"context"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCounter struct {
	value float64
}

func (c *mockCounter) Add(delta float64) {
	c.value += delta
}

func (c *mockCounter) With(labelValues ...string) metrics.Counter {
	return c
}

type mockGauge struct {
	value float64
}

func (g *mockGauge) Add(delta float64) {
	g.value += delta
}

func (g *mockGauge) Set(value float64) {
	g.value = value
}

func (g *mockGauge) With(labelValues ...string) metrics.Gauge {
	return g
}

type mockHistogram struct {
	observations []float64
}

func (h *mockHistogram) Observe(value float64) {
	h.observations = append(h.observations, value)
}

func (h *mockHistogram) With(labelValues ...string) metrics.Histogram {
	return h
}

type mockMetricsProvider struct {
	Counters   map[string]*mockCounter
	Gauges     map[string]*mockGauge
	Histograms map[string]*mockHistogram
}

func newMockMetricsProvider() *mockMetricsProvider {
	return &mockMetricsProvider{
		Counters:   make(map[string]*mockCounter),
		Gauges:     make(map[string]*mockGauge),
		Histograms: make(map[string]*mockHistogram),
	}
}

func (m *mockMetricsProvider) NewCounter(opts metrics.CounterOpts) metrics.Counter {
	c := &mockCounter{}
	m.Counters[opts.Name] = c

	return c
}

func (m *mockMetricsProvider) NewGauge(opts metrics.GaugeOpts) metrics.Gauge {
	g := &mockGauge{}
	m.Gauges[opts.Name] = g

	return g
}

func (m *mockMetricsProvider) NewHistogram(opts metrics.HistogramOpts) metrics.Histogram {
	h := &mockHistogram{}
	m.Histograms[opts.Name] = h

	return h
}

func TestCircuitBreaker(t *testing.T) {
	circuitBreaker := identity.NewCircuitBreaker(identity.CircuitBreakerConfig{
		Threshold: 2,
		Cooldown:  100 * time.Millisecond,
	})

	// Initial state: closed
	assert.True(t, circuitBreaker.Allow())

	// Record one failure
	circuitBreaker.RecordFailure()
	assert.True(t, circuitBreaker.Allow())

	// Record second failure -> opens
	circuitBreaker.RecordFailure()
	assert.False(t, circuitBreaker.Allow())

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)
	assert.True(t, circuitBreaker.Allow())

	// Record success -> resets
	circuitBreaker.RecordFailure()
	circuitBreaker.RecordSuccess()
	circuitBreaker.RecordFailure()
	assert.True(t, circuitBreaker.Allow())
}

func TestProviderObservability(t *testing.T) {
	storage := &mock.Storage{}
	metricsProvider := newMockMetricsProvider()
	p := identity.NewProvider(logging.MustGetLogger(), storage, nil, nil, nil,
		identity.WithMetrics(metricsProvider),
		identity.WithCircuitBreaker(identity.CircuitBreakerConfig{Threshold: 1, Cooldown: time.Hour}),
	)

	// Configure storage to return an error
	storage.StoreIdentityDataReturns(errors.New("storage error"))

	ctx := context.Background()
	data := &driver.RecipientData{Identity: driver.Identity("id")}

	// First call fails
	err := p.RegisterRecipientData(ctx, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")

	// Verify metrics
	assert.InDelta(t, 1.0, metricsProvider.Counters["identity_requests_total"].value, 0.001)
	assert.InDelta(t, 1.0, metricsProvider.Counters["identity_errors_total"].value, 0.001)
	assert.InDelta(t, 0.0, metricsProvider.Gauges["identity_inflight_requests"].value, 0.001)
	assert.Len(t, metricsProvider.Histograms["identity_request_latency_ms"].observations, 1)

	// Circuit should be open now
	err = p.RegisterRecipientData(ctx, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "back-pressure")

	// Verify metrics for the second call
	assert.InDelta(t, 2.0, metricsProvider.Counters["identity_requests_total"].value, 0.001)
	assert.InDelta(t, 2.0, metricsProvider.Counters["identity_errors_total"].value, 0.001)
}
