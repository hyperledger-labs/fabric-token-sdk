/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/driver"
)

type StateDriver interface {
	// NewStateQueryExecutor returns a new StateQueryExecutor for the given URL
	NewStateQueryExecutor(url string) (driver.StateQueryExecutor, error)
	// NewStateVerifier returns a new StateVerifier for the given url
	NewStateVerifier(url string) (driver.StateVerifier, error)
}

type StateDriverName string

type NamedStateDriver struct {
	Name   StateDriverName
	Driver StateDriver
}
