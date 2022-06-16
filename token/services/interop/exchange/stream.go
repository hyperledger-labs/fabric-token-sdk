/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
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

func (o *OutputStream) ByScript() *OutputStream {
	return o.Filter(func(t *token.Output) bool {
		owner, err := identity.UnmarshallRawOwner(t.Owner)
		if err != nil {
			return false
		}
		switch owner.Type {
		case ScriptTypeExchange:
			return true
		}
		return false
	})
}

func (o *OutputStream) ScriptAt(i int) *Script {
	tok := o.OutputStream.At(i)
	owner, err := identity.UnmarshallRawOwner(tok.Owner)
	if err != nil {
		logger.Debugf("failed unmarshalling raw owner [%s]: [%s]", tok, err)
		return nil
	}
	if owner.Type != ScriptTypeExchange {
		logger.Debugf("owner type is [%s] instead of [%s]", owner.Type, ScriptTypeExchange)
		return nil
	}
	script := &Script{}
	if err := json.Unmarshal(owner.Identity, script); err != nil {
		logger.Debugf("failed unmarshalling  exchange script [%s]: [%s]", tok, err)
		return nil
	}
	if script.Sender.IsNone() || script.Recipient.IsNone() {
		return nil
	}
	return script
}
