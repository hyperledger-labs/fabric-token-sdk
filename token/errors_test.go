/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrFailedToGetTMS verifies the error constant is defined
func TestErrFailedToGetTMS(t *testing.T) {
	require.Error(t, ErrFailedToGetTMS)
	assert.Contains(t, ErrFailedToGetTMS.Error(), "failed to get token manager")
}
