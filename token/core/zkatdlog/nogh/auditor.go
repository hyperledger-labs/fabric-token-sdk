/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"context"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type TokenCommitmentLoader interface {
	GetTokenOutputs(ctx context.Context, ids []*token2.ID) (map[string]*token.Token, error)
}

type AuditorService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*v1.PublicParams]
	TokenCommitmentLoader   TokenCommitmentLoader
	Deserializer            driver.Deserializer
	Metrics                 *Metrics
	tracer                  trace.Tracer
}

func NewAuditorService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*v1.PublicParams],
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
		tracer:                  tracerProvider.Tracer("auditor_service", tracing.WithMetricsOpts(tracing.MetricsOpts{Namespace: "nogh"})),
	}
}

// AuditorCheck verifies if the passed tokenRequest matches the tokenRequestMetadata
func (s *AuditorService) AuditorCheck(ctx context.Context, request *driver.TokenRequest, metadata *driver.TokenRequestMetadata, txID string) error {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("start_auditor_check")
	defer span.AddEvent("end_auditor_check")
	s.Logger.Debugf("[%s] check token request validity, number of transfer actions [%d]...", txID, len(metadata.Transfers))

	actionDes := &validator.ActionDeserializer{
		PublicParams: s.PublicParametersManager.PublicParams(),
	}
	_, transfers, err := actionDes.DeserializeActions(request)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize actions")
	}

	tokenIDs := make([]*token2.ID, 0)
	for i, transfer := range metadata.Transfers {
		s.Logger.Debugf("[%s] transfer action [%d] contains [%d] inputs", txID, i, len(transfer.Inputs))
		tokenIDs = append(tokenIDs, transfer.TokenIDs()...)
	}

	span.AddEvent("load_token_outputs")
	// tokenMap, err := s.TokenCommitmentLoader.GetTokenOutputs(ctx, tokenIDs)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed getting token outputs to perform auditor check")
	// }
	s.Logger.Debugf("loaded [%d] corresponding inputs for TX [%s]", len(tokenIDs), txID)

	inputTokens := make([][]*token.Token, len(metadata.Transfers))
	for i, transfer := range transfers {
		if err := transfer.Validate(); err != nil {
			span.AddEvent("failed_to_validate_transfer")
			return errors.Wrapf(err, "failed to validate transfer")
		}
		inputTokens[i] = make([]*token.Token, len(transfer.Inputs))
		for j := range transfer.Inputs {
			inputTokens[i][j] = transfer.Inputs[j].Token
		}
	}

	span.AddEvent("load_public_params")
	pp := s.PublicParametersManager.PublicParams()
	span.AddEvent("create_new_auditor")
	auditor := audit.NewAuditor(s.Logger, s.tracer, s.Deserializer, pp.PedersenGenerators, nil, math.Curves[pp.Curve])
	span.AddEvent("start_auditor_check")
	err = auditor.Check(
		ctx,
		request,
		metadata,
		inputTokens,
		txID,
	)
	if err != nil {
		return errors.WithMessagef(err, "failed to perform auditor check")
	}

	return nil
}
