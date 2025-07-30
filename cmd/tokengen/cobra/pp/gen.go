/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/fabtokenv1"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/zkatdlognoghv1"
	"github.com/spf13/cobra"
)

// GenCmd returns the Cobra Command for Public Params Generation
func GenCmd() *cobra.Command {
	genCobraCommand.AddCommand(fabtokenv1.Cmd())
	genCobraCommand.AddCommand(zkatdlognoghv1.Cmd())

	return genCobraCommand
}

var genCobraCommand = &cobra.Command{
	Use:   "gen",
	Short: "Gen public parameters.",
	Long:  `Generates public parameters.`,
}
