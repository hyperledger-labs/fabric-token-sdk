/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/orion"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/spf13/cobra"
)

// The main command describes the service and defaults to printing the help message.
var mainCmd = &cobra.Command{Use: "orion-helper"}

var (
	initConfig   string
	ppInitConfig string
	network      string
	namespace    string
)

func main() {
	mainCmd.AddCommand(
		newInitCommand(),
		newPPInitCommand(),
	)

	// On failure Cobra prints the usage message and error string, so we only
	// need to exit with a non-0 status
	if mainCmd.Execute() != nil {
		os.Exit(1)
	}
}

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Init DBs and users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := orion.ReadHelperConfig(initConfig)
			if err != nil {
				return err
			}
			return c.InitConfig.Init()
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&initConfig, "config", "c", "", "The path to the init-config file")

	return cmd
}

func newPPInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pp-init",
		Short: "Init PPs for orion",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := orion2.ReadHelperConfig(ppInitConfig)
			if err != nil {
				return err
			}
			hc := c.GetByTMSID(token.TMSID{Network: network, Namespace: namespace})
			if hc == nil {
				return fmt.Errorf("no orion helper found for network %s and namespace %s", network, namespace)
			}
			return hc.Init()
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&ppInitConfig, "config", "c", "", "The path to the pp-init-config file")
	flags.StringVarP(&network, "network", "n", "", "The network to use")
	flags.StringVarP(&namespace, "namespace", "s", "", "The namespace to use")

	return cmd
}
