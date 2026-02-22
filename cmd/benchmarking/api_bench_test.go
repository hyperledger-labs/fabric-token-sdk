/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package benchmarking

import (
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark/node"
	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/stretchr/testify/require"
)

var zkpWorkload = node.Workload{
	Name:    "zkp",
	Factory: &TokenTxVerifyViewFactory{},
	Params:  &TokenTxVerifyParams{},
}

func BenchmarkAPI(b *testing.B) {
	testdataPath := b.TempDir()
	nodeConfPath := path.Join(testdataPath, "fsc", "nodes", "test-node.0")

	// we generate our testdata
	err := node.GenerateConfig(testdataPath)
	require.NoError(b, err)

	// create server
	n, err := node.SetupNode(nodeConfPath, node.NamedFactory{
		Name:    "zkp",
		Factory: &TokenTxVerifyViewFactory{},
	})
	require.NoError(b, err)
	defer n.Stop()

	vm, err := viewregistry.GetManager(n)
	require.NoError(b, err)

	// run workload via direct view API
	node.RunAPIBenchmark(b, vm, zkpWorkload)
}
