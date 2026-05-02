/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package gen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmd tests the Cmd function.
func TestCmd(t *testing.T) {
	cmd := Cmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "artifacts", cmd.Use)

	t.Run("trailing_args", func(t *testing.T) {
		cmd.SetArgs([]string{"extra"})
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trailing args detected")
	})
}

// TestGen tests the gen function and LoadTopologies.
func TestGen(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("fail_no_file", func(t *testing.T) {
		topologyFile = ""
		err := gen(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expecting topology file path")
	})

	t.Run("fail_read_file", func(t *testing.T) {
		topologyFile = "nonexistent.yaml"
		err := gen(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed reading topology file")
	})

	t.Run("fail_unmarshal", func(t *testing.T) {
		topologyFile = filepath.Join(tempDir, "invalid.yaml")
		err := os.WriteFile(topologyFile, []byte("invalid content"), 0644)
		require.NoError(t, err)
		err = gen(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed loading topologies")
	})

	t.Run("success_empty_topologies", func(t *testing.T) {
		topologyFile = filepath.Join(tempDir, "empty.yaml")
		err := os.WriteFile(topologyFile, []byte("topologies: []"), 0644)
		require.NoError(t, err)
		output = tempDir
		port = 20000
		err = gen(nil)
		require.NoError(t, err)
	})

	t.Run("success_with_topology", func(t *testing.T) {
		content := `
topologies:
- type: fabric
  name: fabric
- type: fsc
  name: fsc
- type: token
  name: token
`
		t2, err := LoadTopologies([]byte(content))
		require.NoError(t, err)
		assert.Len(t, t2, 3)
	})

	t.Run("fail_unmarshal_fabric", func(t *testing.T) {
		content := `
topologies:
- type: fabric
  name: [] 
`
		_, err := LoadTopologies([]byte(content))
		require.Error(t, err)
	})
}
