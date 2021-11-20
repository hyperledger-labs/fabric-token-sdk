/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generators

import (
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
)

type fabricPlatform interface {
	DefaultIdemixOrgMSPDir() string
	PeersByID(id string) *fabric.Peer
}

type tokenPlatform interface {
	GetContext() api2.Context
	GetBuilder() api2.Builder
	TokenDir() string
}
