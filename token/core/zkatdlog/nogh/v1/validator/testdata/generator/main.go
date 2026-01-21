package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/testdata"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
)

//go:generate go run . -bits=32,64 -curves=BN254,BLS12_381_BBS_GURVY
func main() {
	flag.Parse()
	// generate setup
	bits, curves, _, err := benchmark2.GenerateCasesWithDefaults()
	if err != nil {
		panic(err)
	}
	configurations, err := benchmark.NewSetupConfigurations("./../../../testdata", bits, curves)
	if err != nil {
		panic(err)
	}
	rootDir := "./../../testdata/"
	if err := configurations.SaveTo(rootDir); err != nil {
		panic(err)
	}

	for k, configuration := range configurations.Configurations {
		// generate the validator env for transfer
		outputDir := filepath.Join(rootDir, k, "transfers")
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			panic(err)
		}

		for i := range 64 {
			env, err := testdata.NewEnv(&benchmark2.Case{
				Bits:       configuration.Bits,
				CurveID:    configuration.CurveID,
				NumInputs:  2,
				NumOutputs: 2,
			}, configurations)
			if err != nil {
				panic(err)
			}
			if err := env.SaveTransferToFile(
				filepath.Join(outputDir, fmt.Sprintf("output.%d.json", i)),
			); err != nil {
				panic(err)
			}
		}
	}
}
