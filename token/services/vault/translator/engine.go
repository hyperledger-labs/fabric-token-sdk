/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package translator

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type Validator interface {
	Verify(ledger token.Ledger, sp token.SignatureProvider, binding string, tr *token.Request) ([]interface{}, error)
}
