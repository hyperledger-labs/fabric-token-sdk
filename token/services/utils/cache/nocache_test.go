/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoCache_Get_Int(t *testing.T) {
	cache := NewNoCache[int]()

	val, found := cache.Get("pineapple")

	require.False(t, found)
	require.Equal(t, 0, val)
}

func TestNoCache_Get_String(t *testing.T) {
	cache := NewNoCache[string]()

	val, found := cache.Get("pineapple")

	require.False(t, found)
	require.Equal(t, "", val)
}

func TestNoCache_GetOrLoad_Success(t *testing.T) {
	cache := NewNoCache[string]()

	loader := func() (string, error) {
		return "loaded", nil
	}

	val, found, err := cache.GetOrLoad("key", loader)

	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, "loaded", val)
}

func TestNoCache_GetOrLoad_Error(t *testing.T) {
	cache := NewNoCache[float64]()

	expectedErr := errors.New("load failed")

	loader := func() (float64, error) {
		return 0.0, expectedErr
	}

	val, found, err := cache.GetOrLoad("key", loader)

	require.ErrorIs(t, err, expectedErr)
	require.False(t, found)
	require.Equal(t, 0.0, val)
}

func TestNoCache_Add_DoesNothing(t *testing.T) {
	cache := NewNoCache[bool]()

	cache.Add("key", true)

	val, found := cache.Get("key")
	require.False(t, found)
	require.Equal(t, false, val)
}

func TestNoCache_Delete_DoesNothing(t *testing.T) {
	cache := NewNoCache[string]()

	cache.Delete("key")

	val, found := cache.Get("key")
	require.False(t, found)
	require.Equal(t, "", val)
}
