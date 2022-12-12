/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
)

type OutputStream struct {
	*token.OutputStream
}

func NewOutputStream(outputs *token.OutputStream) *OutputStream {
	return &OutputStream{OutputStream: outputs}
}

func (o *OutputStream) Filter(f func(t *token.Output) bool) *OutputStream {
	return NewOutputStream(o.OutputStream.Filter(f))
}

func (o *OutputStream) ByRecipient(id view.Identity) *OutputStream {
	return o.Filter(func(t *token.Output) bool {
		return id.Equal(t.Owner)
	})
}

func (o *OutputStream) ByType(typ string) *OutputStream {
	return o.Filter(func(t *token.Output) bool {
		return t.Type == typ
	})
}

func (o *OutputStream) ByScript() *OutputStream {
	return o.Filter(func(t *token.Output) bool {
		owner, err := owner.UnmarshallTypedIdentity(t.Owner)
		if err != nil {
			return false
		}
		return owner.Type == ScriptType
	})
}

func (o *OutputStream) ScriptAt(i int) *Script {
	tok := o.OutputStream.At(i)
	owner, err := owner.UnmarshallTypedIdentity(tok.Owner)
	if err != nil {
		logger.Debugf("failed unmarshalling raw owner [%s]: [%s]", tok, err)
		return nil
	}
	if owner.Type == ScriptType {
		script := &Script{}
		if err := json.Unmarshal(owner.Identity, script); err != nil {
			logger.Debugf("failed unmarshalling pledge script [%s]: [%s]", tok, err)
			return nil
		}
		if script.Sender.IsNone() || script.Recipient.IsNone() {
			return nil
		}
		return script
	}
	return nil
}
