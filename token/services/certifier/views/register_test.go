/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRegisterView_Construction verifies that NewRegisterView stores all
// four parameters correctly and returns a non-nil view.
func TestNewRegisterView_Construction(t *testing.T) {
	v := NewRegisterView("test-network", "test-channel", "test-namespace", "certifier-wallet")

	require.NotNil(t, v)
	assert.Equal(t, "test-network", v.Network)
	assert.Equal(t, "test-channel", v.Channel)
	assert.Equal(t, "test-namespace", v.Namespace)
	assert.Equal(t, "certifier-wallet", v.Wallet)
}

// TestNewRegisterView_EmptyFields verifies that NewRegisterView works with
// empty strings — no panics or nil dereferences on construction.
func TestNewRegisterView_EmptyFields(t *testing.T) {
	v := NewRegisterView("", "", "", "")

	require.NotNil(t, v)
	assert.Empty(t, v.Network)
	assert.Empty(t, v.Channel)
	assert.Empty(t, v.Namespace)
	assert.Empty(t, v.Wallet)
}
