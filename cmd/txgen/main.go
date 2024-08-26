/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/config"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/service/logging"
)

func main() {
	c, err := config.Load()
	if err != nil {
		logging.Logger.Errorf("Could not read config file: %v", err)
		panic(err)
	}

	logging.InitializeLogger(c.App)

	executor, err := txgen.NewSuiteExecutor(c.UserProvider, c.Intermediary, c.Server)
	if err != nil {
		logging.Logger.Errorf("Error creating new runner: %v", err)
		panic(err)
	}
	if err := executor.Execute(c.Suites); err != nil {
		logging.Logger.Errorf("Error initializing executor: %v", err)
		panic(err)
	}
}
