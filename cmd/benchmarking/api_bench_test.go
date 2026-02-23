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

func BenchmarkAPI(b *testing.B) {
	testdataPath := b.TempDir()
	nodeConfPath := path.Join(testdataPath, "fsc", "nodes", "test-node.0")

	err := node.GenerateConfig(testdataPath)
	require.NoError(b, err)

	n, err := node.SetupNode(nodeConfPath, node.NamedFactory{
		Name:    "zkp",
		Factory: &TokenTxVerifyViewFactory{},
	})
	require.NoError(b, err)
	defer n.Stop()

	vm, err := viewregistry.GetManager(n)
	require.NoError(b, err)

	params := &TokenTxVerifyParams{}
	proof, err := GenerateProofData(params)
	require.NoError(b, err)
	params.Proof, err = proof.ToWire()
	require.NoError(b, err)

	wl := node.Workload{
		Name:    "zkp",
		Factory: &TokenTxVerifyViewFactory{},
		Params:  params,
	}

	b.ResetTimer()
	node.RunAPIBenchmark(b, vm, wl)
}
