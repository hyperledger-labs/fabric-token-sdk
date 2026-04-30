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
	"runtime"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/testutils"
	sbenchmark "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemixnym"
)

//go:generate go run . -bits=32,64 -curves=BN254,BLS12_381_BBS_GURVY -num_inputs=1,2 -num_outputs=1,2
func main() {
	flag.Parse()
	// generate setup
	bits, curves, testCases, err := sbenchmark.GenerateCasesWithDefaults()
	if err != nil {
		panic(err)
	}
	configurations, err := benchmark.NewSetupConfigurations("./../../../../testdata", bits, curves, idemixnym.IdentityType)
	if err != nil {
		panic(err)
	}
	rootDir := "./../../zero"
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

			// Create maps to collect all test cases
			transferCases := make(map[string]*testutils.TestCase)
			issueCases := make(map[string]*testutils.TestCase)
			redeemCases := make(map[string]*testutils.TestCase)
			swapCases := make(map[string]*testutils.TestCase)

			// Mutex to protect concurrent map writes
			var mu sync.Mutex

			// Create worker pool with number of CPUs
			numWorkers := runtime.NumCPU()
			taskChan := make(chan int, 64)
			var wg sync.WaitGroup

			// Start workers
			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range taskChan {
						log.Printf("generate [%d]-th env for [bits=%d,curveID=%d,inputs=%d,outputs=%d]...\n",
							i,
							configuration.Bits, configuration.CurveID, testCase.BenchmarkCase.NumInputs, testCase.BenchmarkCase.NumOutputs,
						)
						env, err := testutils.NewEnv(&sbenchmark.Case{
							Bits:       configuration.Bits,
							CurveID:    configuration.CurveID,
							NumInputs:  testCase.BenchmarkCase.NumInputs,
							NumOutputs: testCase.BenchmarkCase.NumOutputs,
						}, configurations)
						if err != nil {
							panic(err)
						}

						// Convert to test cases
						transferCase, err := env.TransferToTestCase()
						if err != nil {
							panic(err)
						}
						issueCase, err := env.IssueToTestCase()
						if err != nil {
							panic(err)
						}
						redeemCase, err := env.RedeemToTestCase()
						if err != nil {
							panic(err)
						}
						swapCase, err := env.SwapToTestCase()
						if err != nil {
							panic(err)
						}

						// Store in maps with mutex protection
						mu.Lock()
						transferCases[fmt.Sprintf("%d", i)] = transferCase
						issueCases[fmt.Sprintf("%d", i)] = issueCase
						redeemCases[fmt.Sprintf("%d", i)] = redeemCase
						swapCases[fmt.Sprintf("%d", i)] = swapCase
						mu.Unlock()
					}
				}()
			}

			// Queue tasks
			for i := range 64 {
				taskChan <- i
			}
			close(taskChan)

			// Wait for all tasks to complete
			wg.Wait()

			// Write aggregated files
			log.Println("writing aggregated files to disk...")
			if err := testutils.SaveAggregatedToFile(filepath.Join(transferOutputDir, "testdata.json"), transferCases); err != nil {
				panic(err)
			}
			if err := testutils.SaveAggregatedToFile(filepath.Join(issueOutputDir, "testdata.json"), issueCases); err != nil {
				panic(err)
			}
			if err := testutils.SaveAggregatedToFile(filepath.Join(redeemOutputDir, "testdata.json"), redeemCases); err != nil {
				panic(err)
			}
			if err := testutils.SaveAggregatedToFile(filepath.Join(swapOutputDir, "testdata.json"), swapCases); err != nil {
				panic(err)
			}
		}
	}
}
