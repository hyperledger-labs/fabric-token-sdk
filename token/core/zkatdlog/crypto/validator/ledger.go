/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package validator

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

//go:generate counterfeiter -o mock/ledger.go -fake-name Ledger . Ledger

type Ledger interface {
	GetState(id token.ID) ([]byte, error)
}
