/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package artifactgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/artifactgen/testdata"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/crypto/zkatdlognoghv1"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTopologies(t *testing.T) {
	tempDir := t.TempDir()
	fileName := filepath.Join(tempDir, "topologies.yaml")
	topologies := testdata.Topology(zkatdlognoghv1.DriverIdentifier, &tokensdk.SDK{})

	err := WriteTopologies(fileName, topologies, 0644)
	require.NoError(t, err)
	assert.FileExists(t, fileName)

	// Test read back
	raw, err := os.ReadFile(fileName)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "topologies:")
}
