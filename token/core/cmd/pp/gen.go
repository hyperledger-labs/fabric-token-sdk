/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	math3 "github.com/IBM/mathlib"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/packager"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
)

const (
	IdemixConfigDirMsp              = "msp"
	IdemixConfigFileIssuerPublicKey = "IssuerPublicKey"
)

var driver string
var idemixMSPDir string
var output string
var base int64
var exponent int
var cc bool

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&driver, "driver", "d", "dlog", "driver (dlog, zkatdlog or fabtoken)")
	flags.StringVarP(&idemixMSPDir, "idemix", "i", "", "idemix msp dir")
	flags.StringVarP(&output, "output", "o", ".", "output folder")
	flags.Int64VarP(&base, "base", "b", 100, "max token quantity")
	flags.IntVarP(&exponent, "exponent", "e", 2, "max token quantity")
	flags.BoolVarP(&cc, "cc", "", false, "generate chaincode package")

	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "gen",
	Short: "Gen zkat artifacts.",
	Long:  `Generates zkat artifacts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		return gen(args)
	},
}

// gen read topology and generates artifacts
func gen(args []string) error {
	var raw []byte
	var err error
	fmt.Printf("Generate public parameters for [%s]...\n", driver)
	switch driver {
	case "dlog", "zkatdlog":
		raw, err = zkatDLogGen(args)
	case "fabtoken":
		raw, err = fabTokenGen(args)
	default:
		errors.Errorf("Invalid crypto type, expected 'dlog, zkatdlog, or fabtoken', got [%s]", driver)
	}
	if err != nil {
		return err
	}

	if cc {
		fmt.Println("Generate chaincode package...")
		if err := genChaincodePackage(raw); err != nil {
			return err
		}
	}

	fmt.Println("Generation done.")
	return nil
}

func zkatDLogGen(args []string) ([]byte, error) {
	// Load Idemix Issuer Public Key
	if len(idemixMSPDir) == 0 {
		return nil, errors.New("identity mixer msp dir is required")
	}
	path := filepath.Join(idemixMSPDir, IdemixConfigDirMsp, IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading idemix issuer public key")
	}

	// Setup
	// TODO: update the curve here
	pp, err := crypto.Setup(base, exponent, ipkBytes, math3.BN254)
	if err != nil {
		return nil, errors.Wrap(err, "failed setting up public parameters")
	}
	// Store Public Params
	raw, err := pp.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing public parameters")
	}
	path = filepath.Join(output, "zkatdlog_pp.json")
	if err := ioutil.WriteFile(path, raw, 0755); err != nil {
		return nil, errors.Wrap(err, "failed writing public parameters to file")
	}

	return raw, nil
}

func fabTokenGen(args []string) ([]byte, error) {
	pp, err := fabtoken.Setup()
	if err != nil {
		return nil, errors.Wrap(err, "failed setting up public parameters")
	}
	// Store Public Params
	raw, err := pp.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing public parameters")
	}
	path := filepath.Join(output, "fabtoken_pp.json")
	if err := ioutil.WriteFile(path, raw, 0755); err != nil {
		return nil, errors.Wrap(err, "failed writing public parameters to file")
	}

	return raw, nil
}

func genChaincodePackage(raw []byte) error {
	t, err := template.New("node").Funcs(template.FuncMap{
		"Params": func() string { return base64.StdEncoding.EncodeToString(raw) },
	}).Parse(DefaultParams)
	if err != nil {
		return errors.Wrap(err, "failed creating params template")
	}
	paramsFile := bytes.NewBuffer(nil)
	err = t.Execute(io.MultiWriter(paramsFile), nil)
	if err != nil {
		return errors.Wrap(err, "failed writing params template")
	}

	err = packager.New().PackageChaincode(
		"github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc/main",
		"golang",
		"tcc",
		filepath.Join(output, "tcc.tar"),
		func(s string, s2 string) (string, []byte) {
			if strings.HasSuffix(s, "github.com/hyperledger-labs/fabric-token-sdk/token/tcc/params.go") {
				return "", paramsFile.Bytes()
			}
			return "", nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed creating chaincode package")
	}

	return nil
}
