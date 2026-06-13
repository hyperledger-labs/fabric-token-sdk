/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
)

// Metrics holds the instrumentation for the auditor Service.
type Metrics struct {
	// AuditDuration is a histogram of the wall-clock time for each Audit()
	// invocation (lock acquisition included), in seconds.
	AuditDuration metrics.Histogram

	// AuditLockConflicts counts calls to Audit() that failed because
	// AcquireLocks returned an error (e.g. contention or timeout).
	AuditLockConflicts metrics.Counter

	// AppendDuration is a histogram of the total wall-clock time for each
	// Append() invocation, in seconds.
	AppendDuration metrics.Histogram

	// AppendErrors counts calls to Append() that failed when writing to the
	// audit database.
	AppendErrors metrics.Counter

	// ReleasesTotal counts all calls to Release(), whether invoked explicitly
	// or via the defer inside Append().
	ReleasesTotal metrics.Counter

	// SetStatusDuration is a histogram of the wall-clock time for each
	// SetStatus() invocation, in seconds.
	SetStatusDuration metrics.Histogram

	// SetStatusErrors counts calls to SetStatus() that returned an error.
	SetStatusErrors metrics.Counter

	// GetStatusDuration is a histogram of the wall-clock time for each
	// GetStatus() invocation, in seconds.
	GetStatusDuration metrics.Histogram

	// GetStatusErrors counts calls to GetStatus() that returned an error.
	GetStatusErrors metrics.Counter

	// GetTokenRequestDuration is a histogram of the wall-clock time for each
	// GetTokenRequest() invocation, in seconds.
	GetTokenRequestDuration metrics.Histogram

	// GetTokenRequestErrors counts calls to GetTokenRequest() that returned an error.
	GetTokenRequestErrors metrics.Counter
}

// newMetrics creates a new Metrics instance with the provided metrics provider.
// If no provider is given, a no-op provider is used that discards all observations.
func newMetrics(p metrics.Provider) *Metrics {
	if p == nil {
		p = &noopProvider{}
	}

	return &Metrics{
		AuditDuration: p.NewHistogram(metrics.HistogramOpts{
			Name:                           "auditor_audit_duration_seconds",
			Help:                           "Histogram of Audit() processing time per transaction (including lock acquisition), in seconds",
			Buckets:                        []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			NativeHistogramBucketFactor:    1.1,
			NativeHistogramMaxBucketNumber: 100,
		}),
		AuditLockConflicts: p.NewCounter(metrics.CounterOpts{
			Name: "auditor_audit_lock_conflicts_total",
			Help: "Total number of Audit() calls that failed to acquire enrollment-ID locks",
		}),
		AppendDuration: p.NewHistogram(metrics.HistogramOpts{
			Name:                           "auditor_append_duration_seconds",
			Help:                           "Histogram of Append() processing time per transaction, in seconds",
			Buckets:                        []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			NativeHistogramBucketFactor:    1.1,
			NativeHistogramMaxBucketNumber: 100,
		}),
		AppendErrors: p.NewCounter(metrics.CounterOpts{
			Name: "auditor_append_errors_total",
			Help: "Total number of Append() calls that failed to write to the audit database",
		}),
		ReleasesTotal: p.NewCounter(metrics.CounterOpts{
			Name: "auditor_releases_total",
			Help: "Total number of Release() calls (explicit and deferred)",
		}),
		SetStatusDuration: p.NewHistogram(metrics.HistogramOpts{
			Name:    "auditor_set_status_duration_seconds",
			Help:    "Histogram of SetStatus() processing time per call, in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),
		SetStatusErrors: p.NewCounter(metrics.CounterOpts{
			Name: "auditor_set_status_errors_total",
			Help: "Total number of SetStatus() calls that returned an error",
		}),
		GetStatusDuration: p.NewHistogram(metrics.HistogramOpts{
			Name:    "auditor_get_status_duration_seconds",
			Help:    "Histogram of GetStatus() processing time per call, in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),
		GetStatusErrors: p.NewCounter(metrics.CounterOpts{
			Name: "auditor_get_status_errors_total",
			Help: "Total number of GetStatus() calls that returned an error",
		}),
		GetTokenRequestDuration: p.NewHistogram(metrics.HistogramOpts{
			Name:    "auditor_get_token_request_duration_seconds",
			Help:    "Histogram of GetTokenRequest() processing time per call, in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),
		GetTokenRequestErrors: p.NewCounter(metrics.CounterOpts{
			Name: "auditor_get_token_request_errors_total",
			Help: "Total number of GetTokenRequest() calls that returned an error",
		}),
	}
}

// noopProvider discards all observations. Used when no provider is configured.
type noopProvider struct{}

// NewCounter creates a no-op counter that discards all observations.
func (p *noopProvider) NewCounter(_ metrics.CounterOpts) metrics.Counter { return &noopCounter{} }

// NewGauge creates a no-op gauge that discards all observations.
func (p *noopProvider) NewGauge(_ metrics.GaugeOpts) metrics.Gauge { return &noopGauge{} }

// NewHistogram creates a no-op histogram that discards all observations.
func (p *noopProvider) NewHistogram(_ metrics.HistogramOpts) metrics.Histogram {
	return &noopHistogram{}
}

type noopCounter struct{}

// With returns the counter itself, ignoring label values.
func (c *noopCounter) With(_ ...string) metrics.Counter { return c }

// Add discards the delta value without recording it.
func (c *noopCounter) Add(delta float64) { _ = delta }

type noopGauge struct{}

// With returns the gauge itself, ignoring label values.
func (g *noopGauge) With(_ ...string) metrics.Gauge { return g }

// Add discards the delta value without recording it.
func (g *noopGauge) Add(delta float64) { _ = delta }

// Set discards the value without recording it.
func (g *noopGauge) Set(val float64) { _ = val }

type noopHistogram struct{}

// With returns the histogram itself, ignoring label values.
func (h *noopHistogram) With(_ ...string) metrics.Histogram { return h }

// Observe discards the value without recording it.
func (h *noopHistogram) Observe(val float64) { _ = val }
