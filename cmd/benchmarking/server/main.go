/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark/node"
	"github.com/hyperledger-labs/fabric-token-sdk/cmd/benchmarking"
)

func main() {
	testdataPath := "./out/testdata" // for local debugging you can set testdataPath := "out/testdata"
	nodeConfPath := path.Join(testdataPath, "fsc", "nodes", "test-node.0")

	// we generate our testdata
	err := node.GenerateConfig(testdataPath)
	if err != nil {
		panic(err)
	}

	// create server
	n, err := node.SetupNode(nodeConfPath, node.NamedFactory{
		Name:    "zkp",
		Factory: &benchmarking.TokenTxVerifyViewFactory{},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Running fscnode %v\n", n.ID())

	// Wait on OS terminate signal.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	n.Stop()

	// cleanup generated data
	_ = os.RemoveAll(testdataPath)
}
