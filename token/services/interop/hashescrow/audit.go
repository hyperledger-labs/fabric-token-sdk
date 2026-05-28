/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
)

type Input struct {
	*token.Input
	isHashEscrow bool
}

func ToInput(i *token.Input) (*Input, error) {
	owner, err := identity.UnmarshalTypedIdentity(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}

	return &Input{
		Input:        i,
		isHashEscrow: owner.Type == HashEscrow,
	}, nil
}

func (i *Input) IsHashEscrow() bool {
	return i.isHashEscrow
}

func (i *Input) Script() (*Script, error) {
	if !i.isHashEscrow {
		return nil, errors.New("this input does not refer to a hash escrow script")
	}

	owner, err := identity.UnmarshalTypedIdentity(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	if owner.Type != HashEscrow {
		return nil, errors.Errorf("invalid identity type, expected [%s], got [%s]", HashEscrow, owner.Type)
	}
	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal hash escrow script")
	}

	return script, nil
}

type Output struct {
	*token.Output
	isHashEscrow bool
}

func ToOutput(i *token.Output) (*Output, error) {
	owner, err := identity.UnmarshalTypedIdentity(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}

	return &Output{
		Output:       i,
		isHashEscrow: owner.Type == HashEscrow,
	}, nil
}

func (o *Output) IsHashEscrow() bool {
	return o.isHashEscrow
}

func (o *Output) Script() (*Script, error) {
	if !o.isHashEscrow {
		return nil, errors.New("this output does not refer to a hash escrow script")
	}

	owner, err := identity.UnmarshalTypedIdentity(o.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	if owner.Type != HashEscrow {
		return nil, errors.Errorf("invalid identity type, expected [%s], got [%s]", HashEscrow, owner.Type)
	}
	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal hash escrow script")
	}

	return script, nil
}
