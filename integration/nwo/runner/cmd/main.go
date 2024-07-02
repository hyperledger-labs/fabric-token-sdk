/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/runner/rpc"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// The main command describes the service and defaults to printing the help message.
var mainCmd = &cobra.Command{Use: "txgen"}

var (
	providerConfigPath string
	suiteConfigPath    string
)

func main() {
	mainCmd.AddCommand(
		startCommand(),
	)

	// On failure Cobra prints the usage message and error string, so we only
	// need to exit with a non-0 status
	if mainCmd.Execute() != nil {
		os.Exit(1)
	}
}

func startCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start txgen",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := os.ReadFile(providerConfigPath)
			if err != nil {
				return err
			}
			var clientConfigs rpc.UserProviderConfig
			if err := yaml.Unmarshal(c, &clientConfigs); err != nil {
				return errors.Wrapf(err, "failed unmarshalling client configs")
			}

			r, err := rpc.NewSuiteExecutor(clientConfigs)
			if err != nil {
				return err
			}

			var suiteConfigs []model.SuiteConfig
			if len(suiteConfigPath) != 0 {
				c, err = os.ReadFile(suiteConfigPath)
				if err != nil {
					return err
				}
				var request struct {
					Suites []model.SuiteConfig `json:"suites"`
				}
				if err := yaml.Unmarshal(c, &request); err != nil {
					return errors.Wrapf(err, "failed unmarshalling suite config")
				}
				suiteConfigs = request.Suites
			}

			if err := r.Execute(suiteConfigs); err != nil {
				return err
			}

			// Wait for execution
			<-make(chan struct{})
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&providerConfigPath, "user-config", "u", "", "The path to the file that specifies the mapping between user and core config")
	flags.StringVarP(&suiteConfigPath, "suite-config", "s", "", "The path to the suite config")

	return cmd
}
