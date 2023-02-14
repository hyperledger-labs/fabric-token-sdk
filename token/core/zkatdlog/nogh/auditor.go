/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// AuditorCheck verifies if the passed tokenRequest matches the tokenRequestMetadata
func (s *Service) AuditorCheck(tokenRequest *driver.TokenRequest, tokenRequestMetadata *driver.TokenRequestMetadata, txID string) error {
	logger.Debugf("[%s] check token request validity, number of transfer actions [%d]...", txID, len(tokenRequestMetadata.Transfers))
	var inputTokens [][]*token.Token
	for i, transfer := range tokenRequestMetadata.Transfers {
		logger.Debugf("[%s] transfer action [%d] contains [%d] inputs", txID, i, len(transfer.TokenIDs))
		inputs, err := s.TokenCommitmentLoader.GetTokenOutputs(transfer.TokenIDs)
		if err != nil {
			return errors.Wrapf(err, "failed getting token outputs to perform auditor check")
		}
		logger.Debugf("[%s] transfer action [%d] contains [%d] inputs, loaded corresponding inputs [%d]", txID, i, len(transfer.TokenIDs), len(inputs))
		inputTokens = append(inputTokens, inputs)
	}

	des, err := s.Deserializer()
	if err != nil {
		return errors.WithMessagef(err, "failed getting deserializer for auditor check")
	}
	pp := s.PublicParams()
	if pp == nil {
		return errors.Errorf("public parameters not inizialized")
	}
	if err := audit.NewAuditor(des, pp.PedParams, pp.IdemixIssuerPK, nil, math.Curves[pp.Curve]).Check(
		tokenRequest,
		tokenRequestMetadata,
		inputTokens,
		txID,
	); err != nil {
		return errors.WithMessagef(err, "failed to perform auditor check")
	}
	return nil
}
