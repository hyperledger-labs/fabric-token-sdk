/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGeneratePackage tests the GeneratePackage function.
func TestGeneratePackage(t *testing.T) {
	t.Run("fail_package", func(t *testing.T) {
		// Providing an invalid path should make PackageChaincode fail.
		err := GeneratePackage([]byte("dummy raw"), "/nonexistent/path/tcc.tar")
		assert.Error(t, err)
	})
}
