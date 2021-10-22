/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package translator

import "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"

//go:generate counterfeiter -o mock/rwset.go -fake-name RWSet . RWSet

// RWSet interface, used to read from, and write to, a rwset.
type RWSet interface {
	driver.RWSet
}
