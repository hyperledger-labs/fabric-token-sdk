/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/pkg/errors"
)

// ScriptInfo includes info about the sender and the recipient
type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}

// GetOwnerAuditInfo returns the audit info of the owner
func GetOwnerAuditInfo(raw []byte, s view2.ServiceProvider) ([]byte, error) {
	if len(raw) == 0 {
		// this is a redeem
		return nil, nil
	}

	owner, err := identity.UnmarshallRawOwner(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal owner of input token")
	}
	if owner.Type == identity.SerializedIdentityType {
		auditInfo, err := view2.GetSigService(s).GetAuditInfo(raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", view.Identity(raw).String())
		}
		return auditInfo, nil
	}
	if owner.Type != exchange.ScriptTypeExchange {
		return nil, errors.Errorf("owner's type not recognized [%s]", owner.Type)
	}
	script := &exchange.Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal RawOwner as an exchange script")
	}

	auditInfo := &ScriptInfo{}
	auditInfo.Sender, err = view2.GetSigService(s).GetAuditInfo(script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for exchange script [%s]", view.Identity(raw).String())
	}

	auditInfo.Recipient, err = view2.GetSigService(s).GetAuditInfo(script.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for exchange script [%s]", view.Identity(raw).String())
	}
	raw, err = json.Marshal(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling audit info for exchange script")
	}
	return raw, nil
}
