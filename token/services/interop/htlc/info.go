/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/pkg/errors"
)

type AuditInfoProvider interface {
	GetAuditInfo(identity token.Identity) ([]byte, error)
}

// ScriptInfo includes info about the sender and the recipient
type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}

func (si *ScriptInfo) Marshal() ([]byte, error) {
	return json.Marshal(si)
}

func (si *ScriptInfo) Unmarshal(raw []byte) error {
	return json.Unmarshal(raw, si)
}

// GetScriptSenderAndRecipient returns the script's sender and recipient according to the type of the given owner
func GetScriptSenderAndRecipient(ro *identity.TypedIdentity) (sender, recipient token.Identity, err error) {
	if ro.Type == ScriptType {
		script := &Script{}
		err = json.Unmarshal(ro.Identity, script)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to unmarshal htlc script")
		}
		return script.Sender, script.Recipient, nil
	}
	return nil, nil, errors.New("unknown identity type")
}
