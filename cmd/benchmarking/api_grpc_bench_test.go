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

// BenchmarkAPIGRPC exercises the ViewAPI via grpc client
func BenchmarkAPIGRPC(b *testing.B) {
	testdataPath := b.TempDir() // for local debugging you can set testdataPath := "out/testdata"
	nodeConfPath := path.Join(testdataPath, "fsc", "nodes", "test-node.0")
	clientConfPath := path.Join(nodeConfPath, "client-config.yaml")

	// we generate our testdata
	err := node.GenerateConfig(testdataPath)
	require.NoError(b, err)

	// create server
	n, err := node.SetupNode(nodeConfPath, node.NamedFactory{
		Name:    "zkp",
		Factory: &TokenTxVerifyViewFactory{},
	})
	require.NoError(b, err)

	// run all workloads via direct view API
	node.RunAPIGRPCBenchmark(b, zkpWorkload,
		clientConfPath, *numConn)

	n.Stop()
}
