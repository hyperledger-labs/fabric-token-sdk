/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/printpp"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	"github.com/spf13/cobra"
)

// UtilsCmd returns the Cobra Command for Public Params Utils command
func UtilsCmd() *cobra.Command {
	utilsCobraCommand.AddCommand(printpp.Cmd())

	return utilsCobraCommand
}

var utilsCobraCommand = &cobra.Command{
	Use:   "pp",
	Short: "Public parameters utils.",
	Long:  `Public parameters utility commands`,
}
