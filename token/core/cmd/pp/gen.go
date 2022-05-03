/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/packager"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	IdemixConfigDirMsp              = "msp"
	IdemixConfigFileIssuerPublicKey = "IssuerPublicKey"
)

var Driver string
var IdemixMSPDir string
var Output string
var Base int64
var Exponent int
var CC bool
var Issuers []string
var Auditors []string

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&Driver, "driver", "d", "dlog", "driver (dlog, zkatdlog or fabtoken)")
	flags.StringVarP(&IdemixMSPDir, "idemix", "i", "", "idemix msp dir")
	flags.StringVarP(&Output, "output", "o", ".", "output folder")
	flags.Int64VarP(&Base, "base", "b", 100, "max token quantity")
	flags.IntVarP(&Exponent, "exponent", "e", 2, "max token quantity")
	flags.BoolVarP(&CC, "cc", "", false, "generate chaincode package")
	flags.StringSliceVarP(&Auditors, "auditors", "a", nil, "list of auditor keys")
	flags.StringSliceVarP(&Issuers, "issuers", "s", nil, "list of issuer keys")

	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "gen",
	Short: "Gen public parameters.",
	Long:  `Generates public parameters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		return Gen(args)
	},
}

// Gen read topology and generates artifacts
func Gen(args []string) error {
	var raw []byte
	var err error
	fmt.Printf("Generate public parameters for [%s]...\n", Driver)
	switch Driver {
	case "dlog", "zkatdlog":
		raw, err = ZKATDLogGen(args)
	case "fabtoken":
		raw, err = FabTokenGen(args)
	default:
		errors.Errorf("Invalid crypto type, expected 'dlog, zkatdlog, or fabtoken', got [%s]", Driver)
	}
	if err != nil {
		return err
	}

	if CC {
		fmt.Println("Generate chaincode package...")
		if err := GenChaincodePackage(raw); err != nil {
			return err
		}
	}

	fmt.Println("Generation done.")
	return nil
}

func ZKATDLogGen(args []string) ([]byte, error) {
	// Load Idemix Issuer Public Key
	if len(IdemixMSPDir) == 0 {
		return nil, errors.New("identity mixer msp dir is required")
	}
	path := filepath.Join(IdemixMSPDir, IdemixConfigDirMsp, IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading idemix issuer public key")
	}

	// Setup
	// TODO: update the curve here
	pp, err := crypto.Setup(Base, Exponent, ipkBytes, math3.BN254)
	if err != nil {
		return nil, errors.Wrap(err, "failed setting up public parameters")
	}
	if err := SetupIssuersAndAuditors(pp); err != nil {
		return nil, err
	}

	// Store Public Params
	raw, err := pp.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing public parameters")
	}
	path = filepath.Join(Output, "zkatdlog_pp.json")
	if err := ioutil.WriteFile(path, raw, 0755); err != nil {
		return nil, errors.Wrap(err, "failed writing public parameters to file")
	}

	return raw, nil
}

func FabTokenGen(args []string) ([]byte, error) {
	// Setup
	pp, err := fabtoken.Setup()
	if err != nil {
		return nil, errors.Wrap(err, "failed setting up public parameters")
	}
	if err := SetupIssuersAndAuditors(pp); err != nil {
		return nil, err
	}
	// Store Public Params
	raw, err := pp.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing public parameters")
	}
	path := filepath.Join(Output, "fabtoken_pp.json")
	if err := ioutil.WriteFile(path, raw, 0755); err != nil {
		return nil, errors.Wrap(err, "failed writing public parameters to file")
	}

	return raw, nil
}

func GenChaincodePackage(raw []byte) error {
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
		"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc/main",
		"golang",
		"tcc",
		filepath.Join(Output, "tcc.tar"),
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

type PP interface {
	AddAuditor(raw view.Identity)
	AddIssuer(raw view.Identity)
}

func SetupIssuersAndAuditors(pp PP) error {
	// Auditors
	for _, auditor := range Auditors {
		// Build an MSP Identity
		entries := strings.Split(auditor, ":")
		if len(entries) != 2 {
			return errors.Errorf("invalid auditor [%s]", auditor)
		}
		provider, err := x509.NewProvider(entries[0], entries[1], nil)
		if err != nil {
			return errors.WithMessagef(err, "failed to create x509 provider for auditor [%s]", auditor)
		}
		id, _, err := provider.Identity(nil)
		if err != nil {
			return errors.WithMessagef(err, "failed to get auditor identity [%s]", auditor)
		}
		pp.AddAuditor(id)
	}
	// Issuers
	for _, issuer := range Issuers {
		// Build an MSP Identity
		entries := strings.Split(issuer, ":")
		if len(entries) != 2 {
			return errors.Errorf("invalid issuer [%s]", issuer)
		}
		provider, err := x509.NewProvider(entries[0], entries[1], nil)
		if err != nil {
			return errors.WithMessagef(err, "failed to create x509 provider for issuer [%s]", issuer)
		}
		id, _, err := provider.Identity(nil)
		if err != nil {
			return errors.WithMessagef(err, "failed to get issuer identity [%s]", issuer)
		}
		pp.AddIssuer(id)
	}
	return nil
}
