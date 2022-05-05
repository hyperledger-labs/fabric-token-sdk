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

var (
	// Driver is the Token-SDK driver to use
	Driver string
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
)

type PP interface {
	AddAuditor(raw view.Identity)
	AddIssuer(raw view.Identity)
}

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&Driver, "driver", "d", "dlog", "driver (dlog, zkatdlog or fabtoken)")
	flags.StringVarP(&IdemixMSPDir, "idemix", "i", "", "idemix msp dir")
	flags.StringVarP(&OutputDir, "output", "o", ".", "output folder")
	flags.BoolVarP(&GenerateCCPackage, "cc", "", false, "generate chaincode package")
	flags.StringSliceVarP(&Auditors, "auditors", "a", nil, "list of auditor keys in the form of <MSP-Dir>:<MSP-ID>")
	flags.StringSliceVarP(&Issuers, "issuers", "s", nil, "list of issuer keys in the form of <MSP-Dir>:<MSP-ID>")

	flags.Int64VarP(&Base, "base", "b", 100, "base field used by the dlog driver")
	flags.IntVarP(&Exponent, "exponent", "e", 2, "exponent field used by the dlog driver")

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

	if GenerateCCPackage {
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
	path = filepath.Join(OutputDir, "zkatdlog_pp.json")
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
	path := filepath.Join(OutputDir, "fabtoken_pp.json")
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
		filepath.Join(OutputDir, "tcc.tar"),
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

func GetMSPIdentity(s string) (view.Identity, error) {
	entries := strings.Split(s, ":")
	if len(entries) != 2 {
		return nil, errors.Errorf("invalid input [%s]", s)
	}
	provider, err := x509.NewProvider(entries[0], entries[1], nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create x509 provider for [%s]", s)
	}
	id, _, err := provider.Identity(nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get identity [%s]", s)
	}
	return id, nil
}

func SetupIssuersAndAuditors(pp PP) error {
	// Auditors
	for _, auditor := range Auditors {
		id, err := GetMSPIdentity(auditor)
		if err != nil {
			return errors.WithMessagef(err, "failed to get auditor identity [%s]", auditor)
		}
		pp.AddAuditor(id)
	}
	// Issuers
	for _, issuer := range Issuers {
		id, err := GetMSPIdentity(issuer)
		if err != nil {
			return errors.WithMessagef(err, "failed to get issuer identity [%s]", issuer)
		}
		pp.AddIssuer(id)
	}
	return nil
}
