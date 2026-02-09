/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/artifactgen/gen"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/certfier"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const CmdRoot = "core"

// The main command describes the service and
// defaults to printing the help message.
var mainCmd = &cobra.Command{Use: "tokengen"}

func main() {
	// For environment variables.
	var unusedVar int

	viper.SetEnvPrefix(CmdRoot)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	// Define command-line flags that are valid for all peer commands and
	// subcommands.
	mainFlags := mainCmd.PersistentFlags()

	mainFlags.String("logging-level", "", "Legacy logging level flag")
	if err := viper.BindPFlag("logging_level", mainFlags.Lookup("logging-level")); err != nil {
		panic(err)
	}
	if err := mainFlags.MarkHidden("logging-level"); err != nil {
		panic(err)
	}

	mainCmd.AddCommand(pp.GenCmd())
	mainCmd.AddCommand(pp.UpdateCmd())
	mainCmd.AddCommand(pp.UtilsCmd())
	mainCmd.AddCommand(certfier.KeyPairGenCmd())
	mainCmd.AddCommand(gen.Cmd())
	mainCmd.AddCommand(version.Cmd())

	// On failure Cobra prints the usage message and error string, so we only
	// need to exit with a non-0 status
	if mainCmd.Execute() != nil {
		os.Exit(1)
	}
}
