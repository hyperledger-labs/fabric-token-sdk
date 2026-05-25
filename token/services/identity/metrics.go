/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
)

// IdentityMetrics contains the metrics for the identity service.
type IdentityMetrics struct {
	// Requests tracks the total number of identity requests.
	Requests metrics.Counter
	// Errors tracks the total number of identity errors.
	Errors metrics.Counter
	// Latency tracks the identity request latency in milliseconds.
	Latency metrics.Histogram
	// InFlight tracks the number of in-flight identity requests.
	InFlight metrics.Gauge
}

// NewIdentityMetrics returns a new instance of IdentityMetrics using the provided metrics provider.
func NewIdentityMetrics(p metrics.Provider) *IdentityMetrics {
	if p == nil {
		return nil
	}

	return &IdentityMetrics{
		Requests: p.NewCounter(metrics.CounterOpts{
			Name: "identity_requests_total",
			Help: "Total number of identity requests.",
		}),
		Errors: p.NewCounter(metrics.CounterOpts{
			Name: "identity_errors_total",
			Help: "Total number of identity errors.",
		}),
		Latency: p.NewHistogram(metrics.HistogramOpts{
			Name:    "identity_request_latency_ms",
			Help:    "Identity request latency in milliseconds.",
			Buckets: []float64{10, 50, 100, 200, 500, 1000, 2000, 5000},
		}),
		InFlight: p.NewGauge(metrics.GaugeOpts{
			Name: "identity_inflight_requests",
			Help: "Number of in-flight identity requests.",
		}),
	}
}
