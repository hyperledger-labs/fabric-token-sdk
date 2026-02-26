/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bench

import (
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark/node"
	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/stretchr/testify/require"
)

func BenchmarkAPI(b *testing.B) {
	testdataPath := b.TempDir()
	nodeConfPath := path.Join(testdataPath, "fsc", "nodes", "test-node.0")

	err := node.GenerateConfig(testdataPath)
	require.NoError(b, err)

	n, err := node.SetupNode(nodeConfPath, node.NamedFactory{
		Name:    "transfer-zkp",
		Factory: &TransferZKViewFactory{},
	})
	require.NoError(b, err)
	defer n.Stop()

	vm, err := viewregistry.GetManager(n)
	require.NoError(b, err)

	wl := node.Workload{
		Name:    "transfer-zkp",
		Factory: &TransferZKViewFactory{},
		Params:  NewTokenTransferVerifyParamsSlice("")[0],
	}

	b.ResetTimer()
	node.RunAPIBenchmark(b, vm, wl)
}
