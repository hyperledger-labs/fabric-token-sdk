/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package auditor exposes internal helpers for use in the auditor_test external test package.
package auditor

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"go.opentelemetry.io/otel/trace"
)

// NewTestMetrics wraps the unexported newMetrics for use in external tests.
func NewTestMetrics(p metrics.Provider) *Metrics { return newMetrics(p) }

// NewTestService creates a Service with all configurable fields for testing.
func NewTestService(
	tmsID token.TMSID,
	networkProvider NetworkProvider,
	auditDB *auditdb.StoreService,
	tokenDB *tokens.Service,
	tmsProvider dep.TokenManagementServiceProvider,
	finalityTracer trace.Tracer,
	metricsProvider metrics.Provider,
	checkService CheckService,
) *Service {
	return &Service{
		tmsID:           tmsID,
		networkProvider: networkProvider,
		auditDB:         auditDB,
		tokenDB:         tokenDB,
		tmsProvider:     tmsProvider,
		finalityTracer:  finalityTracer,
		metricsProvider: metricsProvider,
		metrics:         newMetrics(metricsProvider),
		checkService:    checkService,
	}
}

// RequestWrapperForTest is an exported alias for the unexported requestWrapper type.
type RequestWrapperForTest = requestWrapper

// NewTestRequestWrapper creates a requestWrapper for testing.
func NewTestRequestWrapper(r *token.Request, tms dep.TokenManagementService) *RequestWrapperForTest {
	return newRequestWrapper(r, tms)
}

// CompleteInputsWithEmptyEIDForTest exposes the unexported completeInputsWithEmptyEID for testing.
func CompleteInputsWithEmptyEIDForTest(rw *RequestWrapperForTest, ctx context.Context, record *token.AuditRecord) error {
	return rw.completeInputsWithEmptyEID(ctx, record)
}

// MinimalRequestForTest creates a minimal token.Request for testing.
func MinimalRequestForTest(anchor string) *token.Request {
	return minimalRequest(anchor)
}
