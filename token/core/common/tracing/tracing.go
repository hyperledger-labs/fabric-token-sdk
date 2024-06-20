/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tracing

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"go.opentelemetry.io/otel/trace"
)

var providerType = reflect.TypeOf((*TracerProvider)(nil))

type TracerProvider struct {
	trace.TracerProvider
}

func NewTracerProvider(tp trace.TracerProvider) *TracerProvider {
	return &TracerProvider{TracerProvider: tp}
}

func GetProvider(sp view.ServiceProvider) *TracerProvider {
	s, err := sp.GetService(providerType)
	if err != nil {
		panic(err)
	}
	return s.(*TracerProvider)
}
