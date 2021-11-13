/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/view"

type Driver interface {
	New(sp view.ServiceProvider, network, channel string) (Network, error)
}
