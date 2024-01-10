/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
)

type Input struct {
	*token.Input
	isHTLC   bool
	isPledge bool
}

func ToInput(i *token.Input) (*Input, error) {
	owner, err := owner.UnmarshallTypedIdentity(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	return &Input{
		Input:    i,
		isHTLC:   owner.Type == htlc.ScriptType,
		isPledge: owner.Type == pledge.ScriptType,
	}, nil
}

func (i *Input) IsHTLC() bool {
	return i.isHTLC
}

func (i *Input) IsPledge() bool {
	return i.isPledge
}

func (i *Input) HTLC() (*htlc.Script, error) {
	if !i.isHTLC {
		return nil, errors.New("this input does not refer to an HTLC script")
	}
	owner, err := owner.UnmarshallTypedIdentity(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	script := &htlc.Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal HTLC script")
	}
	return script, nil
}

func (i *Input) Pledge() (*pledge.Script, error) {
	if !i.isPledge {
		return nil, errors.New("this input does not refer to a pledge script")
	}
	owner, err := owner.UnmarshallTypedIdentity(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	script := &pledge.Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal pledge script")
	}
	return script, nil
}

type Output struct {
	*token.Output
	isHTLC   bool
	isPledge bool
}

func ToOutput(o *token.Output) (*Output, error) {
	if o.Owner != nil {
		owner, err := owner.UnmarshallTypedIdentity(o.Owner)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal owner")
		}
		return &Output{
			Output:   o,
			isHTLC:   owner.Type == htlc.ScriptType,
			isPledge: owner.Type == pledge.ScriptType,
		}, nil
	}
	return &Output{
		Output: o,
	}, nil

}

func (o *Output) IsHTLC() bool {
	return o.isHTLC
}

func (o *Output) IsPledge() bool {
	return o.isPledge
}

func (o *Output) HTLC() (*htlc.Script, error) {
	if !o.isHTLC {
		return nil, errors.New("this output does not refer to an HTLC script")
	}
	owner, err := owner.UnmarshallTypedIdentity(o.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	script := &htlc.Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmrshal HTLC script")
	}
	return script, nil
}

func (o *Output) Pledge() (*pledge.Script, error) {
	if !o.isPledge {
		return nil, errors.New("this output does not refer to a pledge script")
	}
	owner, err := owner.UnmarshallTypedIdentity(o.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	script := &pledge.Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmrshal pledge script")
	}
	return script, nil
}
