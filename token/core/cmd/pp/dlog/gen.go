/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"fmt"
	"os"
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
	// GenerateCCPackage indicates whether to generate the chaincode package
	GenerateCCPackage bool
	// Issuers is the list of issuer MSP directories containing the corresponding issuer certificate
	Issuers []string
	// Auditors is the list of auditor MSP directories containing the corresponding auditor certificate
	Auditors []string
	// Base is a dlog driver related parameter
	Base uint
	// Exponent is a dlog driver related parameter
	Exponent uint
	// Aries is a flag to indicate that aries should be used as backend for idemix
	Aries bool
}

var (
	// IdemixMSPDir is the directory containing the Idemix MSP config (Issuer Key Pair)
	IdemixMSPDir string
	// OutputDir is the directory to output the generated files
	OutputDir string
	// GenerateCCPackage indicates whether to generate the chaincode package
	GenerateCCPackage bool
	// Issuers is the list of issuer MSP directories containing the corresponding issuer certificate
	Issuers []string
	// Auditors is the list of auditor MSP directories containing the corresponding auditor certificate
	Auditors []string
	// Base is a dlog driver related parameter.
	// It is used to define the maximum quantity a token can contain as Base^Exponent
	Base uint
	// Exponent is a dlog driver related parameter
	// It is used to define the maximum quantity a token can contain as Base^Exponent
	Exponent uint
	// Aries is a flag to indicate that aries should be used as backend for idemix
	Aries bool
)

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&OutputDir, "output", "o", ".", "output folder")
	flags.BoolVarP(&GenerateCCPackage, "cc", "", false, "generate chaincode package")
	flags.StringSliceVarP(&Auditors, "auditors", "a", nil, "list of auditor MSP directories containing the corresponding auditor certificate")
	flags.StringSliceVarP(&Issuers, "issuers", "s", nil, "list of issuer MSP directories containing the corresponding issuer certificate")
	flags.StringVarP(&IdemixMSPDir, "idemix", "i", "", "idemix msp dir")
	flags.UintVarP(&Base, "base", "b", 100, "base is used to define the maximum quantity a token can contain as Base^Exponent")
	flags.UintVarP(&Exponent, "exponent", "e", 2, "exponent is used to define the maximum quantity a token can contain as Base^Exponent")
	flags.BoolVarP(&Aries, "aries", "r", false, "flag to indicate that aries should be used as backend for idemix")

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
			Aries:             Aries,
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
	curveID := math3.BN254
	if args.Aries {
		curveID = math3.BLS12_381_BBS
	}
	pp, err := crypto.Setup(args.Base, args.Exponent, ipkBytes, curveID)
	if err != nil {
		return nil, errors.Wrap(err, "failed setting up public parameters")
	}
	if err := pp.Validate(); err != nil {
		return nil, errors.Wrapf(err, "failed to validate public parameters")
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
	if err := os.WriteFile(path, raw, 0755); err != nil {
		return nil, errors.Wrap(err, "failed writing public parameters to file")
	}

	return raw, nil
}
