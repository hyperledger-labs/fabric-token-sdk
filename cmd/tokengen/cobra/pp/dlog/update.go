/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// InputFile is the file that contains the public parameters
var InputFile string

type UpdateArgs struct {
	// InputFile is the file that contains the public parameters
	InputFile string
	// OutputDir is the directory to output the generated files
	OutputDir string
	// Issuers is the list of issuer MSP directories containing the corresponding issuer certificate
	Issuers []string
	// Auditors is the list of auditor MSP directories containing the corresponding auditor certificate
	Auditors []string
}

// UpdateCmd returns the Cobra Command for Update
func UpdateCmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cmd.Flags()
	flags.StringVarP(&InputFile, "input", "i", "", "path of the public param file")
	flags.StringVarP(&OutputDir, "output", "o", ".", "output folder")
	flags.StringSliceVarP(&Auditors, "auditors", "a", nil, "list of auditor MSP directories containing the corresponding auditor certificate")
	flags.StringSliceVarP(&Issuers, "issuers", "s", nil, "list of issuer MSP directories containing the corresponding issuer certificate")

	return cmd
}

var cmd = &cobra.Command{
	Use:   "dlog",
	Short: "Update certs in the public parameters file.",
	Long:  "Update certs in the public parameters file without changing the parameters themselves.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		err := Update(&UpdateArgs{
			InputFile: InputFile,
			OutputDir: OutputDir,
			Issuers:   Issuers,
			Auditors:  Auditors,
		})
		if err != nil {
			return errors.Wrap(err, "failed to generate public parameters")
		}
		return nil
	},
}

// Update prints a new version of the config file with updated certs
func Update(args *UpdateArgs) error {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("caught error [%s]\n", e)
		}
	}()
	oldraw, err := os.ReadFile(args.InputFile)
	if err != nil {
		return errors.Wrapf(err, "failed to read input file at [%s]", args.InputFile)
	}

	pp, err := v1.NewPublicParamsFromBytes(oldraw, "zkatdlog")
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal pp from [%s]", args.InputFile)
	}
	if err := pp.Validate(); err != nil {
		return errors.Wrapf(err, "failed to validate public parameters")
	}

	// Clear auditor and issuers if provided, and add them again.
	// If not provided, do not change them.
	if len(args.Auditors) > 0 {
		pp.AuditorIDs = []driver.Identity{}
	}
	if len(args.Issuers) > 0 {
		pp.IssuerIDs = []driver.Identity{}
	}
	if err := common.SetupIssuersAndAuditors(pp, args.Auditors, args.Issuers); err != nil {
		return err
	}

	// Store Public Params
	raw, err := pp.Serialize()
	if err != nil {
		return errors.Wrap(err, "failed serializing public parameters")
	}
	path := filepath.Join(args.OutputDir, "zkatdlog_pp.json")
	if _, err := os.Stat(path); err == nil {
		return errors.New("zkatdlog_pp.json exists in current directory. Specify another output folder with -o")
	}
	if err := os.WriteFile(path, raw, 0755); err != nil {
		return errors.Wrap(err, "failed writing public parameters to file")
	}

	return nil
}
