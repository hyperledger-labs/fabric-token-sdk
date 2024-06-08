/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tracing

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type span struct {
	trace.Span

	start      time.Time
	operations metrics.Counter
	duration   metrics.Histogram
}

func (s *span) End(options ...trace.SpanEndOption) {
	s.Span.End(options...)

	c := trace.NewSpanEndConfig(options...)
	s.updateLabels(c.Attributes())

	s.operations.Add(1)
	s.duration.Observe(c.Timestamp().Sub(s.start).Seconds())
}

func (s *span) AddEvent(name string, options ...trace.EventOption) {
	s.Span.AddEvent(name, options...)

	c := trace.NewEventConfig(options...)
	s.updateLabels(c.Attributes())
}

func (s *span) SetAttributes(kv ...attribute.KeyValue) {
	s.Span.SetAttributes(kv...)

	s.updateLabels(kv)
}

func (s *span) updateLabels(attrs []attribute.KeyValue) {
	labels := make([]string, 0, 2*len(attrs))
	for _, kv := range attrs {
		if kv.Valid() {
			labels = append(labels, string(kv.Key), kv.Value.AsString())
		}
	}

	s.operations = s.operations.With(labels...)
	s.duration = s.duration.With(labels...)
}

func newSpan(backingSpan trace.Span, operations metrics.Counter, delay metrics.Histogram, opts ...trace.SpanStartOption) *span {
	c := trace.NewSpanStartConfig(opts...)
	s := &span{
		Span:       backingSpan,
		start:      defaultNow(c.Timestamp()),
		operations: operations,
		duration:   delay,
	}
	s.updateLabels(c.Attributes())
	return s
}

func defaultNow(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}
