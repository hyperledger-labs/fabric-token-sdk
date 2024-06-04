/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type AuditorService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*crypto.PublicParams]
	TokenCommitmentLoader   TokenCommitmentLoader
	Deserializer            driver.Deserializer
	Metrics                 *Metrics
}

func NewAuditorService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*crypto.PublicParams],
	tokenCommitmentLoader TokenCommitmentLoader,
	deserializer driver.Deserializer,
	metrics *Metrics,
) *AuditorService {
	return &AuditorService{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		TokenCommitmentLoader:   tokenCommitmentLoader,
		Deserializer:            deserializer,
		Metrics:                 metrics,
	}
}

// AuditorCheck verifies if the passed tokenRequest matches the tokenRequestMetadata
func (s *AuditorService) AuditorCheck(tokenRequest *driver.TokenRequest, tokenRequestMetadata *driver.TokenRequestMetadata, txID string) error {
	s.Logger.Debugf("[%s] check token request validity, number of transfer actions [%d]...", txID, len(tokenRequestMetadata.Transfers))
	var inputTokens [][]*token.Token
	for i, transfer := range tokenRequestMetadata.Transfers {
		s.Logger.Debugf("[%s] transfer action [%d] contains [%d] inputs", txID, i, len(transfer.TokenIDs))
		inputs, err := s.TokenCommitmentLoader.GetTokenOutputs(transfer.TokenIDs)
		if err != nil {
			return errors.Wrapf(err, "failed getting token outputs to perform auditor check")
		}
		s.Logger.Debugf("[%s] transfer action [%d] contains [%d] inputs, loaded corresponding inputs [%d]", txID, i, len(transfer.TokenIDs), len(inputs))
		inputTokens = append(inputTokens, inputs)
	}

	pp := s.PublicParametersManager.PublicParams()
	auditor := audit.NewAuditor(
		s.Logger,
		s.Deserializer,
		pp.PedersenGenerators,
		pp.IdemixIssuerPK,
		nil,
		math.Curves[pp.Curve],
	)
	start := time.Now()
	err := auditor.Check(
		tokenRequest,
		tokenRequestMetadata,
		inputTokens,
		txID,
	)
	duration := time.Since(start)
	if err != nil {
		return errors.WithMessagef(err, "failed to perform auditor check")
	}
	s.Metrics.AddAudit()
	s.Metrics.ObserveAuditDuration(duration)

	return nil
}
