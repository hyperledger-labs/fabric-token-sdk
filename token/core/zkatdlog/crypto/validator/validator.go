/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var logger = flogging.MustGetLogger("token-sdk.zkatdlog.validator")

type ValidateTransferFunc = common.ValidateTransferFunc[*crypto.PublicParams, *token.Token, *transfer.TransferAction, *issue.IssueAction]

type ValidateIssueFunc = common.ValidateIssueFunc[*crypto.PublicParams, *token.Token, *transfer.TransferAction, *issue.IssueAction]

type Context = common.Context[*crypto.PublicParams, *token.Token, *transfer.TransferAction, *issue.IssueAction]

type ActionDeserializer struct{}

func (a *ActionDeserializer) DeserializeActions(tr *driver.TokenRequest) ([]*issue.IssueAction, []*transfer.TransferAction, error) {
	issueActions := make([]*issue.IssueAction, len(tr.Issues))
	for i := 0; i < len(tr.Issues); i++ {
		ia := &issue.IssueAction{}
		if err := ia.Deserialize(tr.Issues[i]); err != nil {
			return nil, nil, err
		}
		issueActions[i] = ia
	}

	transferActions := make([]*transfer.TransferAction, len(tr.Transfers))
	for i := 0; i < len(tr.Transfers); i++ {
		ta := &transfer.TransferAction{}
		if err := ta.Deserialize(tr.Transfers[i]); err != nil {
			return nil, nil, err
		}
		transferActions[i] = ta
	}

	return issueActions, transferActions, nil
}

type Validator = common.Validator[*crypto.PublicParams, *token.Token, *transfer.TransferAction, *issue.IssueAction]

func New(pp *crypto.PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferSignatureValidate,
		TransferZKProofValidate,
		TransferHTLCValidate,
	}
	transferValidators = append(transferValidators, extraValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
	}

	return common.NewValidator[*crypto.PublicParams, *token.Token, *transfer.TransferAction, *issue.IssueAction](
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
	)
}
