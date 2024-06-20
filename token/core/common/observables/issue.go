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

type issueMetrics struct {
	issueTracer trace.Tracer
}

func NewIssue(p trace.TracerProvider) *issueMetrics {
	return &issueMetrics{
		issueTracer: p.Tracer("issue", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "token_sdk",
			LabelNames: metrics.AllLabelNames(TokenTypeLabel, SuccessfulLabel),
		})),
	}
}

type ObservableIssueService struct {
	IssueService driver.IssueService
	Metrics      *issueMetrics
}

func NewObservableIssueService(issueService driver.IssueService, metrics *issueMetrics) *ObservableIssueService {
	return &ObservableIssueService{IssueService: issueService, Metrics: metrics}
}

func (o *ObservableIssueService) Issue(issuerIdentity driver.Identity, tokenType string, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, *driver.IssueMetadata, error) {
	_, span := o.Metrics.issueTracer.Start(context.Background(), "issue", trace.WithAttributes(attribute.String(TokenTypeLabel, tokenType)))
	defer span.End()

	action, meta, err := o.IssueService.Issue(issuerIdentity, tokenType, values, owners, opts)
	span.SetAttributes(attribute.Bool(SuccessfulLabel, err == nil))
	return action, meta, err
}

func (o *ObservableIssueService) VerifyIssue(tr driver.IssueAction, metadata [][]byte) error {
	return o.IssueService.VerifyIssue(tr, metadata)
}

func (o *ObservableIssueService) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	return o.IssueService.DeserializeIssueAction(raw)
}
