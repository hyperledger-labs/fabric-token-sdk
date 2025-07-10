/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type ValidateTransferFunc = common.ValidateTransferFunc[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

type ValidateIssueFunc = common.ValidateIssueFunc[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

type ValidateAuditingFunc = common.ValidateAuditingFunc[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

type Context = common.Context[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

type ActionDeserializer struct {
	PublicParams *v1.PublicParams
}

func (a *ActionDeserializer) DeserializeActions(tr *driver.TokenRequest) ([]*issue.Action, []*transfer.Action, error) {
	issueActions := make([]*issue.Action, len(tr.Issues))
	for i := 0; i < len(tr.Issues); i++ {
		ia := &issue.Action{}
		if err := ia.Deserialize(tr.Issues[i]); err != nil {
			return nil, nil, err
		}
		issueActions[i] = ia
	}

	transferActions := make([]*transfer.Action, len(tr.Transfers))
	for i := 0; i < len(tr.Transfers); i++ {
		ta := &transfer.Action{}
		if err := ta.Deserialize(tr.Transfers[i]); err != nil {
			return nil, nil, err
		}
		transferActions[i] = ta
	}

	return issueActions, transferActions, nil
}

type Validator = common.Validator[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

func New(
	logger logging.Logger,
	pp *v1.PublicParams,
	deserializer driver.Deserializer,
	extraTransferValidators []ValidateTransferFunc,
	extraIssuerValidators []ValidateIssueFunc,
	extraAuditorValidators []ValidateAuditingFunc,
) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferActionValidate,
		TransferSignatureValidate,
		TransferUpgradeWitnessValidate,
		TransferZKProofValidate,
		TransferHTLCValidate,
		common.TransferApplicationDataValidate[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer],
	}
	transferValidators = append(transferValidators, extraTransferValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
		common.IssueApplicationDataValidate[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer],
	}
	issueValidators = append(issueValidators, extraIssuerValidators...)

	auditingValidators := []ValidateAuditingFunc{
		common.AuditingSignaturesValidate[*v1.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer],
	}
	auditingValidators = append(auditingValidators, extraAuditorValidators...)

	return common.NewValidator(
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
		auditingValidators,
	)
}
