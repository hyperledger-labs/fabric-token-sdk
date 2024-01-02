/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

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

func (i *Input) Script() (*Script, error) {
	if !i.isHTLC {
		return nil, errors.New("this input does not refer to an HTLC script")
	}

	owner, err := identity.UnmarshallRawOwner(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmrshal HTLC script")
	}
	return script, nil
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
	return &Output{
		Output: i,
		isHTLC: owner.Type == ScriptType,
	}, nil
}

func (o *Output) IsHTLC() bool {
	return o.isHTLC
}

func (o *Output) Script() (*Script, error) {
	if !o.isHTLC {
		return nil, errors.New("this output does not refer to an HTLC script")
	}

	owner, err := identity.UnmarshallRawOwner(o.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmrshal HTLC script")
	}
	return script, nil
}
