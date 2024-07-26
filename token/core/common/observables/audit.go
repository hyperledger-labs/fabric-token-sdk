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

const (
	TokenTypeLabel  metrics.MetricLabel = "token_type"
	SuccessfulLabel metrics.MetricLabel = "successful"
)

type auditMetrics struct {
	auditTracer trace.Tracer
}

func NewAudit(p trace.TracerProvider) *auditMetrics {
	return &auditMetrics{
		auditTracer: p.Tracer("audit", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "token_sdk",
			LabelNames: metrics.AllLabelNames(SuccessfulLabel),
		})),
	}
}

type ObservableAuditorService struct {
	AuditService driver.AuditorService
	Metrics      *auditMetrics
}

func NewObservableAuditorService(auditService driver.AuditorService, metrics *auditMetrics) *ObservableAuditorService {
	return &ObservableAuditorService{AuditService: auditService, Metrics: metrics}
}

func (o *ObservableAuditorService) AuditorCheck(ctx context.Context, request *driver.TokenRequest, metadata *driver.TokenRequestMetadata, anchor string) error {
	newCtx, span := o.Metrics.auditTracer.Start(ctx, "check")
	defer span.End()

	err := o.AuditService.AuditorCheck(newCtx, request, metadata, anchor)
	span.SetAttributes(attribute.Bool(SuccessfulLabel, err == nil))
	return err
}
