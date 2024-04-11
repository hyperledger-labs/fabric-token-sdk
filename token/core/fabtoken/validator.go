/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ValidateTransferFunc = common.ValidateTransferFunc[*PublicParams, *token.Token, *TransferAction, *IssueAction]

type ValidateIssueFunc = common.ValidateIssueFunc[*PublicParams, *token.Token, *TransferAction, *IssueAction]

type ActionDeserializer struct{}

func (a *ActionDeserializer) DeserializeActions(tr *driver.TokenRequest) ([]*IssueAction, []*TransferAction, error) {
	issueActions := make([]*IssueAction, len(tr.Issues))
	for i := 0; i < len(tr.Issues); i++ {
		ia := &IssueAction{}
		if err := ia.Deserialize(tr.Issues[i]); err != nil {
			return nil, nil, err
		}
		issueActions[i] = ia
	}

	transferActions := make([]*TransferAction, len(tr.Transfers))
	for i := 0; i < len(tr.Transfers); i++ {
		ta := &TransferAction{}
		if err := ta.Deserialize(tr.Transfers[i]); err != nil {
			return nil, nil, err
		}
		transferActions[i] = ta
	}

	return issueActions, transferActions, nil
}

type Context = common.Context[*PublicParams, *token.Token, *TransferAction, *IssueAction]

type Validator = common.Validator[*PublicParams, *token.Token, *TransferAction, *IssueAction]

func NewValidator(pp *PublicParams, deserializer driver.Deserializer, extraValidators ...ValidateTransferFunc) *Validator {
	transferValidators := []ValidateTransferFunc{
		TransferSignatureValidate,
		TransferBalanceValidate,
		TransferHTLCValidate,
	}
	transferValidators = append(transferValidators, extraValidators...)

	issueValidators := []ValidateIssueFunc{
		IssueValidate,
	}

	return common.NewValidator[*PublicParams, *token.Token, *TransferAction, *IssueAction](
		logger,
		pp,
		deserializer,
		&ActionDeserializer{},
		transferValidators,
		issueValidators,
		&common.Serializer{},
	)
}
