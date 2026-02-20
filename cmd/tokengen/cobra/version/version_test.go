/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package version

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetInfo tests the GetInfo function.
func TestGetInfo(t *testing.T) {
	info := GetInfo()
	assert.Contains(t, info, ProgramName)
	assert.Contains(t, info, "Version:")
	assert.Contains(t, info, "Go version:")
	assert.Contains(t, info, "OS/Arch:")
}

// TestCmd tests the Cmd function.
func TestCmd(t *testing.T) {
	cmd := Cmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Use)

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, b.String(), ProgramName)

	// Test with trailing args
	cmd.SetArgs([]string{"extra"})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trailing args detected")
}
