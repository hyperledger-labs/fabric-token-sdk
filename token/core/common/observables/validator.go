/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package observables

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type validatorMetrics struct {
	validatorTracer trace.Tracer
}

func NewValidator(p trace.TracerProvider) *validatorMetrics {
	return &validatorMetrics{
		validatorTracer: p.Tracer("validator", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "token_sdk",
			LabelNames: metrics.AllLabelNames(SuccessfulLabel),
		})),
	}
}

type ObservableValidator struct {
	Validator driver.Validator
	Metrics   *validatorMetrics
}

func NewObservableValidator(validator driver.Validator, metrics *validatorMetrics) *ObservableValidator {
	return &ObservableValidator{Validator: validator, Metrics: metrics}
}

func (o *ObservableValidator) UnmarshalActions(raw []byte) ([]interface{}, error) {
	return o.Validator.UnmarshalActions(raw)
}

func (o *ObservableValidator) VerifyTokenRequestFromRaw(ctx context.Context, getState driver.GetStateFnc, anchor string, raw []byte) ([]interface{}, driver.ValidationAttributes, error) {
	newContext, span := o.Metrics.validatorTracer.Start(ctx, "validate")
	defer span.End()

	action, meta, err := o.Validator.VerifyTokenRequestFromRaw(newContext, getState, anchor, raw)
	span.SetAttributes(attribute.Bool(SuccessfulLabel, err == nil))
	return action, meta, err
}
