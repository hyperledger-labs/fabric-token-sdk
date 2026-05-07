/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
)

type Script = hashescrow.Script

// AuditInfoProvider provides access to audit information for a given identity.
//
//go:generate counterfeiter -o mock/aip.go -fake-name AuditInfoProvider . AuditInfoProvider
type AuditInfoProvider interface {
	GetAuditInfo(ctx context.Context, identity driver.Identity) ([]byte, error)
}

// ScriptInfo contains sender and recipient audit info.
type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}
