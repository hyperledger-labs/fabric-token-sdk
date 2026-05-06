/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ethereum

import (
	"errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

const (
	// DriverName identifies the scaffold for an Ethereum/EVM-backed token network driver.
	DriverName = "ethereum"
)

var (
	// ErrNotImplemented marks the parts of the Ethereum driver that are intentionally left for
	// follow-up implementation slices.
	ErrNotImplemented = errors.New("ethereum network driver: not implemented")
)

// Driver is the first scaffold for an Ethereum/EVM token network driver.
//
// This slice only establishes the package boundary and the driver/network types so later PRs can
// build transport, finality, public parameter fetching, and endorsement support incrementally.
type Driver struct{}

// NewDriver returns a new Ethereum driver scaffold.
func NewDriver() *Driver {
	return &Driver{}
}

// New returns a new Ethereum network scaffold for the passed network and channel.
func (d *Driver) New(network, channel string) (driver.Network, error) {
	return NewNetwork(network, channel), nil
}
