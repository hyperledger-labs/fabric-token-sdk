/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type NetworkName string

type NamedDriver struct {
	Name   NetworkName
	Driver Driver
}

// Driver models the network driver factory
type Driver interface {
	// New returns a new network instance for the passed network and channel (if applicable)
	New(network, channel string) (Network, error)
}
