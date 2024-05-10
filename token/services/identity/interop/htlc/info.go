/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/pkg/errors"
)

type AuditInfoProvider interface {
	GetAuditInfo(identity driver.Identity) ([]byte, error)
}

// GetOwnerAuditInfo returns the audit info of the owner
func GetOwnerAuditInfo(raw []byte, s AuditInfoProvider) ([]byte, error) {
	if len(raw) == 0 {
		// this is a redeem
		return nil, nil
	}

	owner, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal owner of input token")
	}

	if owner.Type == htlc.ScriptType {
		sender, recipient, err := GetScriptSenderAndRecipient(owner)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting script sender and recipient")
		}

		auditInfo := &ScriptInfo{}
		auditInfo.Sender, err = s.GetAuditInfo(sender)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for htlc script [%s]", driver.Identity(raw).String())
		}

		auditInfo.Recipient, err = s.GetAuditInfo(recipient)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for script [%s]", driver.Identity(raw).String())
		}
		raw, err = json.Marshal(auditInfo)
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshaling audit info for script")
		}
		return raw, nil
	}

	// delegate
	auditInfo, err := s.GetAuditInfo(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", driver.Identity(raw).String())
	}
	return auditInfo, nil
}

// ScriptInfo includes info about the sender and the recipient
type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}

func (si *ScriptInfo) Marshal() ([]byte, error) {
	return json.Marshal(si)
}

func (si *ScriptInfo) Unarshal(raw []byte) error {
	return json.Unmarshal(raw, si)
}

// GetScriptSenderAndRecipient returns the script's sender and recipient according to the type of the given owner
func GetScriptSenderAndRecipient(ro *identity.TypedIdentity) (sender, recipient driver.Identity, err error) {
	if ro.Type == htlc.ScriptType {
		script := &htlc.Script{}
		err = json.Unmarshal(ro.Identity, script)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to unmarshal htlc script")
		}
		return script.Sender, script.Recipient, nil
	}
	return nil, nil, errors.New("unknown identity type")
}
