/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type ValidateTransferFunc = common.ValidateTransferFunc[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer]

type ValidateIssueFunc = common.ValidateIssueFunc[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer]

type ValidateAuditingFunc = common.ValidateAuditingFunc[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer]

type ActionDeserializer struct{}

func (a *ActionDeserializer) DeserializeActions(tr *driver.TokenRequest) ([]*actions.IssueAction, []*actions.TransferAction, error) {
	issueActions := make([]*actions.IssueAction, len(tr.Issues))
	for i := 0; i < len(tr.Issues); i++ {
		ia := &actions.IssueAction{}
		if err := ia.Deserialize(tr.Issues[i]); err != nil {
			return nil, nil, err
		}
		issueActions[i] = ia
	}

	transferActions := make([]*actions.TransferAction, len(tr.Transfers))
	for i := 0; i < len(tr.Transfers); i++ {
		ta := &actions.TransferAction{}
		if err := ta.Deserialize(tr.Transfers[i]); err != nil {
			return nil, nil, err
		}
		transferActions[i] = ta
	}

	return issueActions, transferActions, nil
}

type Context = common.Context[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer]

type Validator = common.Validator[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer]

func NewValidator(
	logger logging.Logger,
	pp *setup.PublicParams,
	deserializer driver.Deserializer,
	extraTransferValidators []ValidateTransferFunc,
	extraIssuerValidators []ValidateIssueFunc,
	extraAuditorValidators []ValidateAuditingFunc,
) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferActionValidate,
		TransferSignatureValidate,
		TransferBalanceValidate,
		TransferHTLCValidate,
		common.TransferApplicationDataValidate[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer],
	}
	transferValidators = append(transferValidators, extraTransferValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
		common.IssueApplicationDataValidate[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer],
	}
	issueValidators = append(issueValidators, extraIssuerValidators...)

	auditingValidators := []ValidateAuditingFunc{
		common.AuditingSignaturesValidate[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer],
	}
	auditingValidators = append(auditingValidators, extraAuditorValidators...)

	return common.NewValidator[*setup.PublicParams, *actions.Output, *actions.TransferAction, *actions.IssueAction, driver.Deserializer](
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
		auditingValidators,
	)
}
