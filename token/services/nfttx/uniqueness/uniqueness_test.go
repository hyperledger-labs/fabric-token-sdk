/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package uniqueness

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewInMemoryBackend(t *testing.T) {
	backend := NewInMemoryBackend()
	svc := NewService(backend)

	id1, err := svc.ComputeID(map[string]string{"address": "1"})
	require.NoError(t, err)
	require.NotEmpty(t, id1)

	id2, err := svc.ComputeID(map[string]string{"address": "1"})
	require.NoError(t, err)
	require.Equal(t, id1, id2)

	id3, err := svc.ComputeID(map[string]string{"address": "2"})
	require.NoError(t, err)
	require.NotEmpty(t, id3)
	require.NotEqual(t, id1, id3)
}

func TestGetServiceWithBackendOption(t *testing.T) {
	svc := GetService(nil, WithBackend(NewInMemoryBackend()))
	require.NotNil(t, svc)

	id, err := svc.ComputeID("payload")
	require.NoError(t, err)
	require.NotEmpty(t, id)
}
