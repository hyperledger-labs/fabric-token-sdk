/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/cmd"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/cmd/network"
	view "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/client/view/cmd"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/samples/fungible/topology"
	"github.com/pkg/errors"
)

func main() {
	m := cmd.NewMain("Fungible Tokens", "0.1")
	mainCmd := m.Cmd()

	network.StartCMDPostNew = func(infrastructure *integration.Infrastructure) error {
		infrastructure.RegisterPlatformFactory(token.NewPlatformFactory())
		return nil
	}
	network.StartCMDPostStart = func(infrastructure *integration.Infrastructure) error {
		_, err := infrastructure.Client("auditor").CallView("registerAuditor", nil)
		if err != nil {
			return errors.WithMessage(err, "failed to register auditor")
		}
		return nil
	}
	mainCmd.AddCommand(network.NewCmdWithMultipleTopologies(
		map[string][]api.Topology{
			"default": topology.Fabric("dlog"),
			"orion":   topology.Orion("dlog"),
		}))
	mainCmd.AddCommand(view.NewCmd())
	m.Execute()
}
