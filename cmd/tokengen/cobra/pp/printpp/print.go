/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package printpp

import (
	"fmt"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	"github.com/spf13/cobra"
)

var (
	// InputFile is the file that contains the public parameters
	InputFile string
)

type Args struct {
	// InputFile is the file that contains the public parameters
	InputFile string
}

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&InputFile, "input", "i", "", "path of the public param file")

	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "print",
	Short: "Inspect public parameters.",
	Long:  `Inspect public parameters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		err := Print(&Args{
			InputFile: InputFile,
		})
		if err != nil {
			return errors.Wrap(err, "failed to generate public parameters")
		}
		return nil
	},
}

// Print prints the public parameters
func Print(args *Args) error {
	raw, err := os.ReadFile(args.InputFile)
	if err != nil {
		return errors.Wrapf(err, "failed to read file at [%s]", args.InputFile)
	}
	s := core.NewPPManagerFactoryService(fabtoken.NewPPMFactory(), dlog.NewPPMFactory())
	pp, err := s.PublicParametersFromBytes(raw)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal pp from [%s]", args.InputFile)
	}

	fmt.Println(pp.String())

	return nil
}
