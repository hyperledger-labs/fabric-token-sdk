/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/pkg/errors"
)

type Input struct {
	*token.Input
	isHTLC bool
}

func ToInput(i *token.Input) (*Input, error) {
	owner, err := identity.UnmarshallRawOwner(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	return &Input{
		Input:  i,
		isHTLC: owner.Type == ScriptType,
	}, nil
}

func (i *Input) IsHTLC() bool {
	return i.isHTLC
}

type Output struct {
	*token.Output
	isHTLC bool
}

func ToOutput(i *token.Output) (*Output, error) {
	owner, err := identity.UnmarshallRawOwner(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	//// TODO: unmarshal script
	//if owner.Type == ScriptType {
	//	if len(i.OwnerAuditInfo) != 0 {
	//	}
	//}
	return &Output{
		Output: i,
		isHTLC: owner.Type == ScriptType,
	}, nil
}

func (i *Output) IsHTLC() bool {
	return i.isHTLC
}
