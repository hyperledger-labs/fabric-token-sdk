/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration

import (
	"fmt"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/onsi/ginkgo/v2"
)

// TestPortRange represents a port range
type TestPortRange int

const (
	basePort      = 20000
	portsPerNode  = 50
	portsPerSuite = 10 * portsPerNode

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
		LibP2PNoReplication,
		WebSocketWithReplication,
	}
)

const (
	BasePort TestPortRange = basePort + portsPerSuite*iota
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
	FabTokenInteropHTLCTwoFabricNetworks
	FabTokenInteropFastExchangeTwoFabricNetworks
	FabTokenInteropHTLCSwapNoCrossTwoFabricNetworks
	FabTokenInteropHTLCOrion
	FabTokenInteropHTLCSwapNoCrossWithOrionAndFabricNetworks
	ZKATDLogInteropHTLC
	ZKATDLogInteropHTLCTwoFabricNetworks
	ZKATDLogInteropFastExchangeTwoFabricNetworks
	ZKATDLogInteropHTLCSwapNoCrossTwoFabricNetworks
	ZKATDLogInteropHTLCOrion
	ZKATDLogInteropHTLCSwapNoCrossWithOrionAndFabricNetworks
	Mixed
)

// StartPortForNode On linux, the default ephemeral port range is 32768-60999 and can be
// allocated by the system for the client side of TCP connections or when
// programs explicitly request one. Given linux is our default CI system,
// we want to try avoid ports in that range.
func (t TestPortRange) StartPortForNode() int {
	const startEphemeral, endEphemeral = 32768, 60999

	port := int(t) + portsPerNode*(ginkgo.GinkgoParallelProcess()-1)
	if port >= startEphemeral-portsPerNode && port <= endEphemeral-portsPerNode {
		fmt.Fprintf(os.Stderr, "WARNING: port %d is part of the default ephemeral port range on linux", port)
	}
	return port
}
