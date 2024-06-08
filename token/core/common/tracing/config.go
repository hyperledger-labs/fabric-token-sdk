/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tracing

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	namespaceKey    = "namespace"
	labelNamesKey   = "label_names"
	statsdFormatKey = "statsd_format"
)

type MetricsOpts struct {
	Namespace    string
	LabelNames   []string
	StatsdFormat string
}

func WithMetricsOpts(o MetricsOpts) trace.TracerOption {
	set := attribute.NewSet(
		attribute.String(namespaceKey, o.Namespace),
		attribute.StringSlice(labelNamesKey, o.LabelNames),
		attribute.String(statsdFormatKey, o.StatsdFormat),
	)
	return trace.WithInstrumentationAttributes(set.ToSlice()...)
}

func extractMetricsOpts(attrs attribute.Set) MetricsOpts {
	o := MetricsOpts{}
	if val, ok := attrs.Value(namespaceKey); ok {
		o.Namespace = val.AsString()
	}
	if val, ok := attrs.Value(labelNamesKey); ok {
		o.LabelNames = val.AsStringSlice()
	}
	if val, ok := attrs.Value(statsdFormatKey); ok {
		o.StatsdFormat = val.AsString()
	}
	return o
}
