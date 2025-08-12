/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package zkatdlognoghv1

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/crypto/zkatdlognoghv1"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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
	// Version allows the caller of tokengen to override the version number put in the public params
	Version uint
}

// UpdateCmd returns the Cobra Command for Update
func UpdateCmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cmd.Flags()
	flags.StringVarP(&InputFile, "input", "i", "", "path of the public param file")
	flags.StringVarP(&OutputDir, "output", "o", ".", "output folder")
	flags.StringSliceVarP(&Auditors, "auditors", "a", nil, "list of auditor MSP directories containing the corresponding auditor certificate")
	flags.StringSliceVarP(&Issuers, "issuers", "s", nil, "list of issuer MSP directories containing the corresponding issuer certificate")
	flags.UintVarP(&Version, "version", "v", 0, "allows the caller of tokengen to override the version number put in the public params")

	return cmd
}

var cmd = &cobra.Command{
	Use:   zkatdlognoghv1.DriverIdentifier,
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
			Version:   Version,
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

	pp, err := v1.NewPublicParamsFromBytes(oldraw, v1.DLogNoGHDriverName, v1.ProtocolV1)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal pp from [%s]", args.InputFile)
	}
	if err := pp.Validate(); err != nil {
		return errors.Wrapf(err, "failed to validate public parameters")
	}

	// Clear auditor and issuers if provided, and add them again.
	// If not provided, do not change them.
	if len(args.Auditors) > 0 {
		pp.SetAuditors(nil)
	}
	if len(args.Issuers) > 0 {
		pp.SetIssuers(nil)
	}
	if err := common.SetupIssuersAndAuditors(pp, args.Auditors, args.Issuers); err != nil {
		return err
	}

	// update version if needed
	ver := v1.ProtocolV1
	if args.Version != 0 {
		ver = driver.TokenDriverVersion(args.Version)
	}
	pp.DriverVersion = ver

	// Store Public Params
	raw, err := pp.Serialize()
	if err != nil {
		return errors.Wrap(err, "failed serializing public parameters")
	}
	fileName := fmt.Sprintf("zkatdlognoghv%d_pp.json", ver)
	path := filepath.Join(
		args.OutputDir,
		fileName,
	)
	if _, err := os.Stat(path); err == nil {
		return errors.Errorf("%s exists in current directory. Specify another output folder with -o", fileName)
	}
	if err := os.WriteFile(path, raw, 0755); err != nil {
		return errors.Wrap(err, "failed writing public parameters to file")
	}

	return nil
}
