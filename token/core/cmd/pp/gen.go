/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/fabtoken"
	"github.com/spf13/cobra"
)

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	cobraCommand.AddCommand(fabtoken.Cmd())
	cobraCommand.AddCommand(dlog.Cmd())

	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "gen",
	Short: "Gen public parameters.",
	Long:  `Generates public parameters.`,
}
