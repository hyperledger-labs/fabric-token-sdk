/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	api3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

func (s *Service) AuditorCheck(tokenRequest *api3.TokenRequest, tokenRequestMetadata *api3.TokenRequestMetadata, txID string) error {
	logger.Debugf("check token request validity...")
	var inputTokens [][]*token.Token
	for _, transfer := range tokenRequestMetadata.Transfers {
		inputs, err := s.TokenCommitmentLoader.GetTokenCommitments(transfer.TokenIDs)
		if err != nil {
			return errors.Wrapf(err, "failed getting token commitments")
		}
		inputTokens = append(inputTokens, inputs)
	}

	pp := s.PublicParams()
	auditor, err := audit.NewAuditor(pp, nil)
	if err != nil {
		return errors.WithMessagef(err, "failed instantiating auditor")
	}
	if err := auditor.Check(
		tokenRequest,
		tokenRequestMetadata,
		inputTokens,
		txID,
	); err != nil {
		return errors.WithMessagef(err, "failed checking transaction")
	}
	return nil
}
