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
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type AuditorService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*crypto.PublicParams]
	TokenCommitmentLoader   TokenCommitmentLoader
	Deserializer            driver.Deserializer
	Metrics                 *Metrics
	tracer                  trace.Tracer
}

func NewAuditorService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*crypto.PublicParams],
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
	newCtx, span := s.tracer.Start(ctx, "auditor_check")
	defer span.End()
	s.Logger.Debugf("[%s] check token request validity, number of transfer actions [%d]...", txID, len(metadata.Transfers))
	var inputTokens [][]*token.Token
	span.AddEvent("check_token_validity")
	for i, transfer := range metadata.Transfers {
		s.Logger.Debugf("[%s] transfer action [%d] contains [%d] inputs", txID, i, len(transfer.TokenIDs))
		inputs, err := s.TokenCommitmentLoader.GetTokenOutputs(transfer.TokenIDs)
		if err != nil {
			return errors.Wrapf(err, "failed getting token outputs to perform auditor check")
		}
		s.Logger.Debugf("[%s] transfer action [%d] contains [%d] inputs, loaded corresponding inputs [%d]", txID, i, len(transfer.TokenIDs), len(inputs))
		inputTokens = append(inputTokens, inputs)
	}

	span.AddEvent("load_public_params")
	pp := s.PublicParametersManager.PublicParams()
	span.AddEvent("create_new_auditor")
	auditor := audit.NewAuditor(
		s.Logger,
		s.tracer,
		s.Deserializer,
		pp.PedersenGenerators,
		pp.IdemixIssuerPK,
		nil,
		math.Curves[pp.Curve],
	)
	span.AddEvent("start_auditor_check")
	err := auditor.Check(
		newCtx,
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
