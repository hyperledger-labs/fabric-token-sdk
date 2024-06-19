/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tracing

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/tracing"
	metrics2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"go.opentelemetry.io/otel/trace"
)

type metricsProvider interface {
	NewCounter(opts metrics2.CounterOpts) metrics2.Counter
	NewHistogram(opts metrics2.HistogramOpts) metrics2.Histogram
}

func NewProvider(metricsProvider) trace.TracerProvider {

	p, _ := tracing.NoopProvider()
	return p
}
