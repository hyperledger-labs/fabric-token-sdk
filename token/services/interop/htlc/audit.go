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

func InputToScript(i *token.Input) (*Script, error) {
	owner, err := identity.UnmarshallRawOwner(i.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	if owner.Type != ScriptType {
		return nil, nil
	}
	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, err
	}
	return script, nil
}

func OutputToScript(o *token.Output) (*Script, error) {
	owner, err := identity.UnmarshallRawOwner(o.Owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal owner")
	}
	if owner.Type != ScriptType {
		return nil, nil
	}
	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, err
	}
	return script, nil
}
