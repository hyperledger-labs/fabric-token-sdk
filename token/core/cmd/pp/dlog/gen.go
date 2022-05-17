/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/cc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type GeneratorArgs struct {
	// IdemixMSPDir is the directory containing the Idemix MSP config (Issuer Key Pair)
	IdemixMSPDir string
	// OutputDir is the directory to output the generated files
	OutputDir string
	// GenerateCCPackage is whether to generate the chaincode package
	GenerateCCPackage bool
	// Issuers is the list of issuers to include in the public parameters.
	// Each issuer should be specified in the form of <MSP-Dir>:<MSP-ID>
	Issuers []string
	// Auditors is the list of auditors to include in the public parameters.
	// Each auditor should be specified in the form of <MSP-Dir>:<MSP-ID>
	Auditors []string
	// Base is a dlog driver related parameter
	Base int64
	// Exponent is a dlog driver related parameter
	Exponent int
}

var (
	// IdemixMSPDir is the directory containing the Idemix MSP config (Issuer Key Pair)
	IdemixMSPDir string
	// OutputDir is the directory to output the generated files
	OutputDir string
	// GenerateCCPackage is whether to generate the chaincode package
	GenerateCCPackage bool
	// Issuers is the list of issuers to include in the public parameters.
	// Each issuer should be specified in the form of <MSP-Dir>:<MSP-ID>
	Issuers []string
	// Auditors is the list of auditors to include in the public parameters.
	// Each auditor should be specified in the form of <MSP-Dir>:<MSP-ID>
	Auditors []string
	// Base is a dlog driver related parameter.
	// It is used to define the maximum quantity a token can contain as Base^Exponent
	Base int64
	// Exponent is a dlog driver related parameter
	// It is used to define the maximum quantity a token can contain as Base^Exponent
	Exponent int
)

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&OutputDir, "output", "o", ".", "output folder")
	flags.BoolVarP(&GenerateCCPackage, "cc", "", false, "generate chaincode package")
	flags.StringSliceVarP(&Auditors, "auditors", "a", nil, "list of auditor keys in the form of <MSP-Dir>:<MSP-ID>")
	flags.StringSliceVarP(&Issuers, "issuers", "s", nil, "list of issuer keys in the form of <MSP-Dir>:<MSP-ID>")
	flags.StringVarP(&IdemixMSPDir, "idemix", "i", "", "idemix msp dir")
	flags.Int64VarP(&Base, "base", "b", 100, "base is used to define the maximum quantity a token can contain as Base^Exponent")
	flags.IntVarP(&Exponent, "exponent", "e", 2, "exponent is used to define the maximum quantity a token can contain as Base^Exponent")

	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "dlog",
	Short: "Gen ZKAT DLog public parameters.",
	Long:  `Generates ZKAT DLog public parameters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		raw, err := Gen(&GeneratorArgs{
			IdemixMSPDir:      IdemixMSPDir,
			OutputDir:         OutputDir,
			GenerateCCPackage: GenerateCCPackage,
			Issuers:           Issuers,
			Auditors:          Auditors,
			Base:              Base,
			Exponent:          Exponent,
		})
		if err != nil {
			return errors.Wrap(err, "failed to generate public parameters")
		}
		// generate the chaincode package
		if GenerateCCPackage {
			fmt.Println("Generate chaincode package...")
			if err := cc.GeneratePackage(raw, OutputDir); err != nil {
				return err
			}
		}
		return nil
	},
}

// Gen generates the public parameters for the ZKATDLog driver
func Gen(args *GeneratorArgs) ([]byte, error) {
	// Load Idemix Issuer Public Key
	_, ipkBytes, err := idemix.LoadIssuerPublicKey(args.IdemixMSPDir)
	if err != nil {
		return nil, err
	}

	// Setup
	// TODO: update the curve here
	pp, err := crypto.Setup(args.Base, args.Exponent, ipkBytes, math3.BN254)
	if err != nil {
		return nil, errors.Wrap(err, "failed setting up public parameters")
	}
	if err := common.SetupIssuersAndAuditors(pp, args.Auditors, args.Issuers); err != nil {
		return nil, err
	}

	// Store Public Params
	raw, err := pp.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing public parameters")
	}
	path := filepath.Join(args.OutputDir, "zkatdlog_pp.json")
	if err := ioutil.WriteFile(path, raw, 0755); err != nil {
		return nil, errors.Wrap(err, "failed writing public parameters to file")
	}

	return raw, nil
}
