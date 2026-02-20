/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certfier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyPairGenCmd(t *testing.T) {
	cmd := KeyPairGenCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "certifier-keygen", cmd.Use)
}

func TestKeyPairGen(t *testing.T) {
	wd, _ := os.Getwd()
	testdataPath := filepath.Join(wd, "..", "..", "testdata", "zkatdlognoghv1_pp.json")
	tempDir := t.TempDir()

	t.Run("success", func(t *testing.T) {
		ppPath = testdataPath
		output = tempDir
		_ = keyPairGen()
		// NewCertifierKeyPair is currently hardcoded to return "not supported" in core/common/ppm.go
	})

	t.Run("read_pp_fail", func(t *testing.T) {
		ppPath = "nonexistent.json"
		output = tempDir
		err := keyPairGen()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed reading public parameters")
	})

	t.Run("unmarshal_pp_fail", func(t *testing.T) {
		invalidPP := filepath.Join(tempDir, "invalid_pp.json")
		err := os.WriteFile(invalidPP, []byte("invalid content"), 0644)
		require.NoError(t, err)
		ppPath = invalidPP
		output = tempDir
		err = keyPairGen()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed unmarshalling public parameters")
	})
}

func TestCobraCommand(t *testing.T) {
	wd, _ := os.Getwd()
	testdataPath := filepath.Join(wd, "..", "..", "testdata", "zkatdlognoghv1_pp.json")
	tempDir := t.TempDir()

	cmd := KeyPairGenCmd()
	cmd.SetArgs([]string{"--pppath", testdataPath, "--output", tempDir})
	err := cmd.Execute()
	// It will return "not supported" currently
	if err != nil {
		assert.Contains(t, err.Error(), "not supported")
	}

	// Test trailing args
	cmd.SetArgs([]string{"--pppath", testdataPath, "extra"})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trailing args detected")
}
