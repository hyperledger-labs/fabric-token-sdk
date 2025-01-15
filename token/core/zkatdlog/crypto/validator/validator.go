/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type ValidateTransferFunc = common.ValidateTransferFunc[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

type ValidateIssueFunc = common.ValidateIssueFunc[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

type Context = common.Context[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

type ActionDeserializer struct{}

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

type Validator = common.Validator[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer]

func New(logger logging.Logger, pp *crypto.PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferActionValidate,
		TransferSignatureValidate,
		TransferUpgradeWitnessValidate,
		TransferZKProofValidate,
		TransferHTLCValidate,
	}
	transferValidators = append(transferValidators, extraValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
	}

	return common.NewValidator[*crypto.PublicParams, *token.Token, *transfer.Action, *issue.Action, driver.Deserializer](
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
		&common.Serializer{},
	)
}
