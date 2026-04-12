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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

type TokenCommitmentLoader interface {
	GetTokenOutputs(ctx context.Context, ids []*token2.ID) (map[string]*token.Token, error)
}

type AuditorService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*setup.PublicParams]
	TokenCommitmentLoader   TokenCommitmentLoader
	Deserializer            driver.Deserializer
	Metrics                 *Metrics
	tracer                  trace.Tracer
}

func NewAuditorService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*setup.PublicParams],
	tokenCommitmentLoader TokenCommitmentLoader,
	deserializer driver.Deserializer,
	metrics *Metrics,
	tracerProvider trace.TracerProvider,
) *AuditorService {
	return &AuditorService{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		TokenCommitmentLoader:   tokenCommitmentLoader,
		Deserializer:            deserializer,
		Metrics:                 metrics,
		tracer:                  tracerProvider.Tracer("auditor_service", tracing.WithMetricsOpts(tracing.MetricsOpts{})),
	}
}

// AuditorCheck verifies if the passed tokenRequest matches the tokenRequestMetadata
func (s *AuditorService) AuditorCheck(ctx context.Context, request *driver.TokenRequest, metadata *driver.TokenRequestMetadata, anchor driver.TokenRequestAnchor) error {
	s.Logger.DebugfContext(ctx, "[%s] check token request validity, number of transfer actions [%d]...", anchor, len(metadata.Transfers))

	actionDes := &validator.ActionDeserializer{
		PublicParams: s.PublicParametersManager.PublicParams(),
	}
	_, transfers, err := actionDes.DeserializeActions(request)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize actions")
	}

	// Collect token IDs from the deserialized actions so we always have non-nil IDs,
	// independent of whether the caller populated metadata.Transfers[i].Inputs[j].TokenID.
	tokenIDs := make([]*token2.ID, 0)
	for i, transfer := range transfers {
		s.Logger.DebugfContext(ctx, "[%s] transfer action [%d] contains [%d] inputs", anchor, i, len(transfer.Inputs))
		for _, input := range transfer.Inputs {
			tokenIDs = append(tokenIDs, input.ID)
		}
	}

	// Load the actual token commitments from the ledger.  ctx carries any deadline
	// set by the caller so the retry loop inside GetTokenOutputs respects it.
	tokenMap, err := s.TokenCommitmentLoader.GetTokenOutputs(ctx, tokenIDs)
	if err != nil {
		return errors.Wrapf(err, "failed getting token outputs to perform auditor check for [%s]", anchor)
	}
	s.Logger.DebugfContext(ctx, "loaded [%d] ledger tokens for TX [%s]", len(tokenMap), anchor)

	// Build the per-transfer, per-input token slices from the ledger map.
	// Using the action's input IDs as keys preserves positional correspondence
	// with metadata.Transfers[i].Inputs[j] which GetAuditInfoForTransfers relies on.
	inputTokens := make([][]*token.Token, len(transfers))
	for i, transfer := range transfers {
		if err := transfer.Validate(); err != nil {
			s.Logger.ErrorfContext(ctx, "failed to validate transfer: %s", err)

			return errors.Wrapf(err, "failed to validate transfer")
		}
		inputTokens[i] = make([]*token.Token, len(transfer.Inputs))
		for j, input := range transfer.Inputs {
			tok, ok := tokenMap[input.ID.String()]
			if !ok {
				return errors.Errorf("ledger token [%v] not found for transfer [%d] input [%d] in TX [%s]", input.ID, i, j, anchor)
			}
			inputTokens[i][j] = tok
		}
	}

	pp := s.PublicParametersManager.PublicParams()
	auditor := audit.NewAuditor(s.Logger, s.tracer, s.Deserializer, pp.PedersenGenerators, math.Curves[pp.Curve])
	s.Logger.DebugfContext(ctx, "Start auditor check")
	err = auditor.Check(
		ctx,
		request,
		metadata,
		inputTokens,
		anchor,
	)
	if err != nil {
		return errors.WithMessagef(err, "failed to perform auditor check")
	}

	return nil
}
