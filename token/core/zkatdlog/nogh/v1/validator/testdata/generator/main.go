/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/testdata"
	sbenchmark "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
)

//go:generate go run . -bits=32,64 -curves=BN254,BLS12_381_BBS_GURVY -num_inputs=1,2 -num_outputs=1,2
func main() {
	flag.Parse()
	// generate setup
	bits, curves, testCases, err := sbenchmark.GenerateCasesWithDefaults()
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
		for _, testCase := range testCases {
			transferOutputDir := filepath.Join(rootDir, k, fmt.Sprintf("transfers_i%d_o%d", testCase.BenchmarkCase.NumInputs, testCase.BenchmarkCase.NumOutputs))
			issueOutputDir := filepath.Join(rootDir, k, fmt.Sprintf("issues_i%d_o%d", testCase.BenchmarkCase.NumInputs, testCase.BenchmarkCase.NumOutputs))
			redeemOutputDir := filepath.Join(rootDir, k, fmt.Sprintf("redeems_i%d_o%d", testCase.BenchmarkCase.NumInputs, testCase.BenchmarkCase.NumOutputs))
			swapOutputDir := filepath.Join(rootDir, k, fmt.Sprintf("swaps_i%d_o%d", testCase.BenchmarkCase.NumInputs, testCase.BenchmarkCase.NumOutputs))

			for _, path := range []string{transferOutputDir, redeemOutputDir, swapOutputDir, issueOutputDir} {
				if err := os.MkdirAll(path, 0o755); err != nil {
					panic(err)
				}
			}

			for i := range 64 {
				log.Printf("generate [%d]-th env for [bits=%d,curveID=%d,inputs=%d,outputs=%d]...\n",
					i,
					configuration.Bits, configuration.CurveID, testCase.BenchmarkCase.NumInputs, testCase.BenchmarkCase.NumOutputs,
				)
				env, err := testdata.NewEnv(&sbenchmark.Case{
					Bits:       configuration.Bits,
					CurveID:    configuration.CurveID,
					NumInputs:  testCase.BenchmarkCase.NumInputs,
					NumOutputs: testCase.BenchmarkCase.NumOutputs,
				}, configurations)
				if err != nil {
					panic(err)
				}

				log.Println("store to disk...")
				if err := env.SaveTransferToFile(filepath.Join(transferOutputDir, fmt.Sprintf("output.%d.json", i))); err != nil {
					panic(err)
				}
				if err := env.SaveIssueToFile(filepath.Join(issueOutputDir, fmt.Sprintf("output.%d.json", i))); err != nil {
					panic(err)
				}
				if err := env.SaveRedeemToFile(filepath.Join(redeemOutputDir, fmt.Sprintf("output.%d.json", i))); err != nil {
					panic(err)
				}
				if err := env.SaveSwapToFile(filepath.Join(swapOutputDir, fmt.Sprintf("output.%d.json", i))); err != nil {
					panic(err)
				}
			}
		}
	}
}
