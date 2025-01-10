/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/onsi/ginkgo/v2"
)

// TestPortRange represents a port range
type TestPortRange integration.TestPortRange

const (
	SimpleTokenSelector    = "simple"
	SherdLockTokenSelector = "sherdlock"
)

var (
	TokenSelectors = []string{SimpleTokenSelector, SherdLockTokenSelector}
)

type InfrastructureType struct {
	Label             ginkgo.Labels
	CommType          fsc.P2PCommunicationType
	ReplicationFactor int
}

var (
	WebSocketNoReplication = &InfrastructureType{
		Label:             ginkgo.Label("websocket"),
		CommType:          fsc.WebSocket,
		ReplicationFactor: token.None,
	}
	LibP2PNoReplication = &InfrastructureType{
		Label:             ginkgo.Label("libp2p"),
		CommType:          fsc.LibP2P,
		ReplicationFactor: token.None,
	}
	WebSocketWithReplication = &InfrastructureType{
		Label:             ginkgo.Label("replicas"),
		CommType:          fsc.WebSocket,
		ReplicationFactor: 3,
	}

	WebSocketNoReplicationOnly = []*InfrastructureType{
		WebSocketNoReplication,
	}
	LibP2PNoReplicationOnly = []*InfrastructureType{
		LibP2PNoReplication,
	}
	WebSocketWithReplicationOnly = []*InfrastructureType{
		WebSocketWithReplication,
	}

	AllTestTypes = []*InfrastructureType{
		WebSocketNoReplication,
		// LibP2PNoReplication,
		// WebSocketWithReplication,
	}
)

const (
	BasePort integration.TestPortRange = integration.BasePort + integration.PortsPerSuite*iota

	ZKATDLogFungible
	ZKATDLogFungibleStress
	ZKATDLogFungibleHSM

	FabTokenFungible

	OrionZKATDLogBasics
	OrionFabTokenBasics

	ZKATDLogDVP
	FabTokenDVP

	ZKATDLogNFT
	FabTokenNFT

	OrionZKATDLogNFT
	OrionFabTokenNFT

	FabTokenInteropHTLC
	FabTokenInteropHTLCOrion
	FabTokenInteropHTLCTwoFabricNetworks
	FabTokenInteropHTLCSwapNoCrossTwoFabricNetworks
	FabTokenInteropHTLCSwapNoCrossWithOrionAndFabricNetworks

	ZKATDLogInteropHTLC
	ZKATDLogInteropHTLCTwoFabricNetworks
	ZKATDLogInteropHTLCSwapNoCrossTwoFabricNetworks
	ZKATDLogInteropHTLCOrion
	ZKATDLogInteropHTLCSwapNoCrossWithOrionAndFabricNetworks

	Mixed
)
