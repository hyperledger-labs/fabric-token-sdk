/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package drivers

import (
	"reflect"
	"sort"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type Holder[D any] struct {
	logger    logging.Logger
	driversMu sync.RWMutex
	Drivers   map[DriverName]D
}

func NewHolder[D any]() *Holder[D] {
	return &Holder[D]{
		logger:  logging.MustGetLogger("token-sdk.manager.drivers"),
		Drivers: make(map[DriverName]D),
	}
}

func (h *Holder[D]) Get(name DriverName) (D, bool) {
	d, ok := h.Drivers[name]
	return d, ok
}

// Register makes a DB driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func (h *Holder[D]) Register(name DriverName, driver D) {
	h.driversMu.Lock()
	defer h.driversMu.Unlock()
	// This will fail for non nil-able inputs, but passing such services would be a programming error detected on start up
	if v := reflect.ValueOf(driver); v.IsNil() {
		panic("Register driver is nil")
	}
	if _, dup := h.Drivers[name]; dup {
		panic("Register called twice for driver " + name)
	}
	h.Drivers[name] = driver
}

// DriverNames returns a sorted list of the names of the registered drivers.
func (h *Holder[D]) DriverNames() []string {
	h.driversMu.RLock()
	defer h.driversMu.RUnlock()
	list := make([]string, 0, len(h.Drivers))
	for name := range h.Drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}
