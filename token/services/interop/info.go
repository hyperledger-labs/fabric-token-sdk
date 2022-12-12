/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
)

type AuditInfoProvider interface {
	GetAuditInfo(identity view.Identity) ([]byte, error)
}

// GetOwnerAuditInfo returns the audit info of the owner
func GetOwnerAuditInfo(raw []byte, s AuditInfoProvider) ([]byte, error) {
	if len(raw) == 0 {
		// this is a redeem
		return nil, nil
	}

	identity, err := owner.UnmarshallTypedIdentity(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal owner")
	}
	if identity.Type == owner.SerializedIdentityType {
		auditInfo, err := s.GetAuditInfo(raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for identity [%s]", view.Identity(raw).String())
		}
		return auditInfo, nil
	}

	sender, recipient, issuer, err := GetScriptSenderAndRecipient(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting script sender and recipient")
	}

	auditInfo := &ScriptInfo{}

	if identity.Type == htlc.ScriptType {
		auditInfo.Sender, err = s.GetAuditInfo(sender)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for sender of htlc script [%s]", view.Identity(raw).String())
		}

		auditInfo.Recipient, err = s.GetAuditInfo(recipient)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for recipient of htlc script [%s]", view.Identity(raw).String())
		}
	}

	if identity.Type == pledge.ScriptType {
		auditInfo.Sender, err = s.GetAuditInfo(sender)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for sender of pledge script [%s]", view.Identity(raw).String())
		}

		if len(auditInfo.Sender) == 0 { // in case this is a redeem we need to check the script issuer (and not the script sender)
			auditInfo.Sender, err = s.GetAuditInfo(issuer)
			if err != nil {
				return nil, errors.Wrapf(err, "failed getting audit info for issuer of pledge script [%s]", view.Identity(raw).String())
			}
			if len(auditInfo.Sender) == 0 {
				return nil, errors.Errorf("failed getting audit info for pledge script [%s]", view.Identity(raw).String())
			}
		}

		// Notice that recipient is in another network, but the issuer is
		// the actual recipient of the script because it is in the same network.
		auditInfo.Recipient, err = s.GetAuditInfo(issuer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for issuer of pledge script [%s]", view.Identity(raw).String())
		}
	}

	raw, err = json.Marshal(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling audit info for script")
	}
	return raw, nil
}

// ScriptInfo includes info about the sender and the recipient
type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}

// GetScriptSenderAndRecipient returns the script's sender, recipient, and issuer, according to the type of the given owner
func GetScriptSenderAndRecipient(ro *owner.TypedIdentity) (sender, recipient, issuer view.Identity, err error) {
	if ro.Type == htlc.ScriptType {
		script := &htlc.Script{}
		err = json.Unmarshal(ro.Identity, script)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to unmarshal htlc script")
		}
		return script.Sender, script.Recipient, nil, nil
	}
	if ro.Type == pledge.ScriptType {
		script := &pledge.Script{}
		err = json.Unmarshal(ro.Identity, script)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to unmarshal pledge script")
		}
		return script.Sender, script.Recipient, script.Issuer, nil
	}
	return nil, nil, nil, errors.Errorf("owner's type not recognized [%s]", ro.Type)
}
