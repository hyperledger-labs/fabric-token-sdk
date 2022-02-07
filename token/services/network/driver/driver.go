/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/view"

// Driver models the network driver factory
type Driver interface {
	// New returns a new network instance for the passed network and channel (if applicable)
	New(sp view.ServiceProvider, network, channel string) (Network, error)
}
