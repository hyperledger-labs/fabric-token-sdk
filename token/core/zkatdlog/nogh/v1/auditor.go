/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

type AuditorService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*setup.PublicParams]
	Deserializer            driver.Deserializer
	QueryEngine             driver.QueryEngine
	tracer                  trace.Tracer
}

func NewAuditorService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*setup.PublicParams],
	deserializer driver.Deserializer,
	queryEngine driver.QueryEngine,
	tracerProvider trace.TracerProvider,
) *AuditorService {
	return &AuditorService{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		Deserializer:            deserializer,
		QueryEngine:             queryEngine,
		tracer:                  tracerProvider.Tracer("auditor_service", tracing.WithMetricsOpts(tracing.MetricsOpts{})),
	}
}

// AuditorCheck verifies if the passed tokenRequest matches the tokenRequestMetadata
func (s *AuditorService) AuditorCheck(ctx context.Context, request *driver.TokenRequest, metadata *driver.TokenRequestMetadata, anchor driver.TokenRequestAnchor) error {
	s.Logger.DebugfContext(ctx, "[%s] check token request validity, number of transfer actions [%d]...", anchor, metadata.NumTransfers())

	// Extract all TokenIDs from both transfer and issue actions in metadata and check for duplicates
	tokenIDMap := make(map[string]*token.ID)
	var tokenIDs []*token.ID

	for i, action := range metadata.Actions {
		// Extract TokenIDs from transfer actions
		if action.TransferMetadata != nil {
			ids := action.TransferMetadata.TokenIDs()
			for _, id := range ids {
				if id == nil {
					continue
				}
				// Check for duplicates using string representation as key
				idKey := id.String()
				if _, exists := tokenIDMap[idKey]; exists {
					return errors.Errorf("duplicate token ID [%s] found in metadata at action index [%d] for tx [%s]", idKey, i, anchor)
				}
				tokenIDMap[idKey] = id
				tokenIDs = append(tokenIDs, id)
			}
		}

		// Extract TokenIDs from issue action inputs (for token upgrades/conversions)
		if action.IssueMetadata != nil {
			for _, input := range action.IssueMetadata.Inputs {
				if input == nil || input.TokenID == nil {
					continue
				}
				id := input.TokenID
				// Check for duplicates using string representation as key
				idKey := id.String()
				if _, exists := tokenIDMap[idKey]; exists {
					return errors.Errorf("duplicate token ID [%s] found in metadata at action index [%d] for tx [%s]", idKey, i, anchor)
				}
				tokenIDMap[idKey] = id
				tokenIDs = append(tokenIDs, id)
			}
		}
	}

	// Retrieve audit tokens from the query engine
	var auditTokens map[*token.ID]*token.Token
	if len(tokenIDs) > 0 {
		s.Logger.DebugfContext(ctx, "[%s] retrieving [%d] audit tokens...", anchor, len(tokenIDs))
		tokens, err := s.QueryEngine.ListAuditTokens(ctx, tokenIDs...)
		if err != nil {
			return errors.WithMessagef(err, "failed to retrieve audit tokens for tx [%s]", anchor)
		}

		// Build the token map
		auditTokens = make(map[*token.ID]*token.Token, len(tokens))
		for i, tok := range tokens {
			if tok != nil {
				auditTokens[tokenIDs[i]] = tok
			}
		}
		s.Logger.DebugfContext(ctx, "[%s] retrieved [%d] audit tokens", anchor, len(auditTokens))
	}

	pp := s.PublicParametersManager.PublicParams()
	auditor := audit.NewAuditor(s.Logger, s.tracer, s.Deserializer, pp.PedersenGenerators, math.Curves[pp.Curve])
	s.Logger.DebugfContext(ctx, "Start auditor check")
	err := auditor.Check(
		ctx,
		request,
		metadata,
		anchor,
		auditTokens,
	)
	if err != nil {
		return errors.WithMessagef(err, "failed to perform auditor check")
	}

	return nil
}
