/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionKeeper(t *testing.T) {
	vk := &VersionKeeper{}

	// initially version 0
	require.Equal(t, uint64(0), vk.GetVersion())

	for i := range uint64(6) {
		vk.UpdateVersion()
		require.Equal(t, i, vk.GetVersion())
	}
}
