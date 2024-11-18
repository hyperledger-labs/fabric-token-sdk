/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type ValidateTransferFunc = common.ValidateTransferFunc[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer]

type ValidateIssueFunc = common.ValidateIssueFunc[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer]

type Context = common.Context[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer]

type ActionDeserializer struct{}

func (a *ActionDeserializer) DeserializeActions(tr *driver.TokenRequest) ([]*issue.IssueAction, []*transfer.Action, error) {
	issueActions := make([]*issue.IssueAction, len(tr.Issues))
	for i := 0; i < len(tr.Issues); i++ {
		ia := &issue.IssueAction{}
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

type Validator = common.Validator[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer]

func New(logger logging.Logger, pp *crypto.PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferSignatureValidate,
		TransferZKProofValidate,
		htlc.TransferHTLCValidate[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer],
		pledge.TransferPledgeValidate[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer],
	}
	transferValidators = append(transferValidators, extraValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
		pledge.IssuePledgeValidate[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer],
	}

	return common.NewValidator[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.IssueAction, driver.Deserializer](
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
		&common.Serializer{},
	)
}
