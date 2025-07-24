/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certfier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/zkatdlognoghv1"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/core"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	dlogdriver "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var driver string
var ppPath string
var output string

// KeyPairGenCmd returns the Cobra Command for the Certifier KeyGen
func KeyPairGenCmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&driver, "driver", "d", zkatdlognoghv1.DriverIdentifier, "driver (dlog)")
	flags.StringVarP(&ppPath, "pppath", "p", "", "path to the public parameters file")
	flags.StringVarP(&output, "output", "o", ".", "output folder")

	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "certifier-keygen",
	Short: "Gen Token Certifier Key Pair.",
	Long:  `Gen Token Certifier Key Pair.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		return keyPairGen(args)
	},
}

// keyPairGen
func keyPairGen(args []string) error {
	// TODO:
	// 1. load public parameters from ppPath
	fmt.Printf("Read public parameters from file [%s]...\n", ppPath)
	ppRaw, err := os.ReadFile(ppPath)
	if err != nil {
		return errors.Wrapf(err, "failed reading public parameters from [%s]", ppPath)
	}
	s := driver2.NewPPManagerFactoryService(fabtoken.NewPPMFactory(), dlogdriver.NewPPMFactory())
	pp, err := s.PublicParametersFromBytes(ppRaw)
	if err != nil {
		return errors.Wrapf(err, "failed unmarshalling public parameters loaded from [%s], len [%d]", ppPath, len(ppRaw))
	}
	ppm, err := s.NewPublicParametersManager(pp)
	if err != nil {
		return errors.Wrapf(err, "failed instantiating public parameters manager")
	}
	// ppm.PublicParameters().GraphHiding()

	// 2. generate certifier key-pair
	fmt.Printf("Generate certifier key-pair...\n")
	skRaw, pkRaw, err := ppm.NewCertifierKeyPair()
	if err != nil {
		return errors.Wrapf(err, "failed generating new certifier key-pair")
	}

	// 3. store key-pair under output
	if err := os.MkdirAll(output, 0766); err != nil {
		return errors.Wrap(err, "failed making output dir")
	}
	skPath := filepath.Join(output, "certifier.sk")
	pkPath := filepath.Join(output, "certifier.pk")
	fmt.Printf("Store key-pair to [%s,%s]...\n", skPath, pkPath)
	if err := os.WriteFile(skPath, skRaw, 0600); err != nil {
		return errors.Wrap(err, "failed writing certifier secret key to file")
	}
	if err := os.WriteFile(pkPath, pkRaw, 0600); err != nil {
		return errors.Wrap(err, "failed writing certifier public key to file")
	}

	return nil
}
