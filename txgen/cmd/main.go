/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger-labs/fabric-token-sdk/txgen"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/config"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/service/logging"
)

func main() {
	c, err := config.Load()
	if err != nil {
		logging.Logger.Errorf("Could not read config file: %v", err)
		panic(err)
	}

	logging.InitializeLogger(c.App)

	container, err := txgen.NewRunner(c.UserProvider, c.Intermediary)
	if err != nil {
		logging.Logger.Errorf("Error creating new runner: %v", err)
		panic(err)
	}

	if err = container.SuiteRunner.Run(c.Suites); err != nil {
		logging.Logger.Errorf("Error happened during run of the program: %s", err.GetMessage())
		panic(err)
	}
}
