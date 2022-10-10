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
	logger.Debugf("check token request validity...")
	var inputTokens [][]*token.Token
	for _, transfer := range tokenRequestMetadata.Transfers {
		inputs, err := s.TokenCommitmentLoader.GetTokenCommitments(transfer.TokenIDs)
		if err != nil {
			return errors.Wrapf(err, "failed getting token commitments to perform auditor check")
		}
		inputTokens = append(inputTokens, inputs)
	}

	des, err := s.Deserializer()
	if err != nil {
		return errors.WithMessagef(err, "failed getting deserializer for auditor check")
	}
	pp := s.PublicParams()
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
