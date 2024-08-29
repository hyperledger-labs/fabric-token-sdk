/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/fabtoken"
	"github.com/spf13/cobra"
)

// GenCmd returns the Cobra Command for Public Params Generation
func GenCmd() *cobra.Command {
	genCobraCommand.AddCommand(fabtoken.Cmd())
	genCobraCommand.AddCommand(dlog.Cmd())

	return genCobraCommand
}

var genCobraCommand = &cobra.Command{
	Use:   "gen",
	Short: "Gen public parameters.",
	Long:  `Generates public parameters.`,
}
