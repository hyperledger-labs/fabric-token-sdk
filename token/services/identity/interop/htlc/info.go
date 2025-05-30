/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type AuditInfoProvider interface {
	GetAuditInfo(ctx context.Context, identity driver.Identity) ([]byte, error)
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
