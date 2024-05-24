/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/driver"
)

type StateDriver interface {
	// NewStateQueryExecutor returns a new StateQueryExecutor for the given URL
	NewStateQueryExecutor(sp token.ServiceProvider, url string) (driver.StateQueryExecutor, error)
	// NewStateVerifier returns a new StateVerifier for the given url
	NewStateVerifier(sp token.ServiceProvider, url string) (driver.StateVerifier, error)
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]StateDriver)
)

// RegisterStateDriver makes an SSPDriver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func RegisterStateDriver(name string, driver StateDriver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("Register called twice for driver " + name)
	}
	drivers[name] = driver
}
