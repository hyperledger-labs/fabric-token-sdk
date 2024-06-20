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

var providerType = reflect.TypeOf((*tracerProvider)(nil))

type tracerProvider struct {
	trace.TracerProvider
}

func NewTracerProvider(tp trace.TracerProvider) trace.TracerProvider {
	return &tracerProvider{TracerProvider: tp}
}

func Get(sp view.ServiceProvider) trace.TracerProvider {
	s, err := sp.GetService(providerType)
	if err != nil {
		panic(err)
	}
	return s.(*tracerProvider)
}
