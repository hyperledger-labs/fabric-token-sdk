/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/zkatdlognoghv1"
	"github.com/spf13/cobra"
)

// UpdateCmd returns the Cobra Command for updating the config file
func UpdateCmd() *cobra.Command {
	// not implemented for fabtoken
	updateCobraCommand.AddCommand(zkatdlognoghv1.UpdateCmd())

	return updateCobraCommand
}

var updateCobraCommand = &cobra.Command{
	Use:   "update",
	Short: "Update certs in the public parameters file.",
	Long:  "Update auditor and issuer certs in the public parameters file without changing the parameters themselves.",
}
