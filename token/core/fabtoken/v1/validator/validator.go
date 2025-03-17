/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type ValidateTransferFunc = common.ValidateTransferFunc[*core.PublicParams, *core.Output, *core.TransferAction, *core.IssueAction, driver.Deserializer]

type ValidateIssueFunc = common.ValidateIssueFunc[*core.PublicParams, *core.Output, *core.TransferAction, *core.IssueAction, driver.Deserializer]

type ActionDeserializer struct{}

func (a *ActionDeserializer) DeserializeActions(tr *driver.TokenRequest) ([]*core.IssueAction, []*core.TransferAction, error) {
	issueActions := make([]*core.IssueAction, len(tr.Issues))
	for i := 0; i < len(tr.Issues); i++ {
		ia := &core.IssueAction{}
		if err := ia.Deserialize(tr.Issues[i]); err != nil {
			return nil, nil, err
		}
		issueActions[i] = ia
	}

	transferActions := make([]*core.TransferAction, len(tr.Transfers))
	for i := 0; i < len(tr.Transfers); i++ {
		ta := &core.TransferAction{}
		if err := ta.Deserialize(tr.Transfers[i]); err != nil {
			return nil, nil, err
		}
		transferActions[i] = ta
	}

	return issueActions, transferActions, nil
}

type Context = common.Context[*core.PublicParams, *core.Output, *core.TransferAction, *core.IssueAction, driver.Deserializer]

type Validator = common.Validator[*core.PublicParams, *core.Output, *core.TransferAction, *core.IssueAction, driver.Deserializer]

func NewValidator(logger logging.Logger, pp *core.PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferActionValidate,
		TransferSignatureValidate,
		TransferBalanceValidate,
		TransferHTLCValidate,
	}
	transferValidators = append(transferValidators, extraValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
	}

	return common.NewValidator[*core.PublicParams, *core.Output, *core.TransferAction, *core.IssueAction, driver.Deserializer](
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
	)
}
