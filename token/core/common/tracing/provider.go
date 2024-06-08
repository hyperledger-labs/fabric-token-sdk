/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tracing

import (
	"fmt"

	metrics2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

type metricsProvider interface {
	NewCounter(opts metrics2.CounterOpts) metrics2.Counter
	NewHistogram(opts metrics2.HistogramOpts) metrics2.Histogram
}

func NewProvider(metricsProvider metricsProvider) trace.TracerProvider {
	//lint:ignore SA1019 This implementation is faster than the current one, and we don't need to copy the code
	return NewProviderWithBackingProvider(metricsProvider, trace.NewNoopTracerProvider())
}

func NewProviderWithBackingProvider(metricsProvider metricsProvider, backingProvider trace.TracerProvider) trace.TracerProvider {
	return &tracerProvider{
		metricsProvider: metricsProvider,
		backingProvider: backingProvider,
	}
}

type tracerProvider struct {
	embedded.TracerProvider

	metricsProvider metricsProvider
	backingProvider trace.TracerProvider
}

func (p *tracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	c := trace.NewTracerConfig(options...)

	opts := extractMetricsOpts(c.InstrumentationAttributes())
	return &tracer{
		backingTracer: p.backingProvider.Tracer(name, options...),
		operations: p.metricsProvider.NewCounter(metrics.CounterOpts{
			Namespace:    opts.Namespace,
			Name:         fmt.Sprintf("%s_operations", name),
			Help:         fmt.Sprintf("Counter of '%s' operations", name),
			LabelNames:   opts.LabelNames,
			StatsdFormat: opts.StatsdFormat,
		}),
		duration: p.metricsProvider.NewHistogram(metrics.HistogramOpts{
			Namespace:    opts.Namespace,
			Name:         fmt.Sprintf("%s_duration", name),
			Help:         fmt.Sprintf("Histogram for the duration of '%s' operations", name),
			LabelNames:   opts.LabelNames,
			StatsdFormat: opts.StatsdFormat,
		}),
	}
}
