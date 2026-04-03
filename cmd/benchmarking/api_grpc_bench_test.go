/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmarking

import (
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/integration/benchmark/node"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/benchmark/flags"
)

var numConn = flags.IntSlice("numConn", []int{1, 2, 4, 8}, "Number of grpc client connections - may be a comma-separated list")

// BenchmarkAPIGRPC exercises the ViewAPI via grpc client.
// The proof is pre-computed and embedded in the workload params so every
// gRPC request carries it.
func BenchmarkAPIGRPC(b *testing.B) {
	testdataPath := b.TempDir()
	nodeConfPath := path.Join(testdataPath, "fsc", "nodes", "test-node.0")
	clientConfPath := path.Join(nodeConfPath, "client-config.yaml")

	err := node.GenerateConfig(testdataPath)
	require.NoError(b, err)

	n, err := node.SetupNode(nodeConfPath, node.NamedFactory{
		Name:    "zkp",
		Factory: &TokenTxVerifyViewFactory{},
	})
	require.NoError(b, err)
	defer n.Stop()

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

	node.RunAPIGRPCBenchmark(b, wl, clientConfPath, *numConn)
}
