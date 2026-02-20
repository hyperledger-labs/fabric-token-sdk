/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadIssuerPublicKey(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("success", func(t *testing.T) {
		mspDir := filepath.Join(tempDir, "msp")
		err := os.MkdirAll(mspDir, 0750)
		require.NoError(t, err)
		ipkPath := filepath.Join(mspDir, "IssuerPublicKey")
		err = os.WriteFile(ipkPath, []byte("dummy ipk"), 0644)
		require.NoError(t, err)

		path, content, err := LoadIssuerPublicKey(tempDir)
		require.NoError(t, err)
		assert.Equal(t, ipkPath, path)
		assert.Equal(t, []byte("dummy ipk"), content)
	})

	t.Run("fail_not_found", func(t *testing.T) {
		_, _, err := LoadIssuerPublicKey("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed reading idemix issuer public key")
	})
}
