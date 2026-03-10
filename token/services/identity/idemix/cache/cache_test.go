/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

// TestIdentityCache verifies basic cache functionality and identity retrieval.
func TestIdentityCache(t *testing.T) {
	c := NewIdentityCache(func(context.Context, []byte) (*idriver.IdentityDescriptor, error) {
		return &idriver.IdentityDescriptor{
			Identity:  []byte("hello world"),
			AuditInfo: []byte("audit"),
		}, nil
	}, 100, nil, NewMetrics(&disabled.Provider{}))
	identityDescriptor, err := c.Identity(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, driver.Identity([]byte("hello world")), identityDescriptor.Identity)
	assert.Equal(t, []byte("audit"), identityDescriptor.AuditInfo)

	identityDescriptor, err = c.Identity(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, driver.Identity([]byte("hello world")), identityDescriptor.Identity)
	assert.Equal(t, []byte("audit"), identityDescriptor.AuditInfo)
}

// TestIdentityCacheForRace tests concurrent cache access for thread-safety.
func TestIdentityCacheForRace(t *testing.T) {
	c := NewIdentityCache(func(context.Context, []byte) (*idriver.IdentityDescriptor, error) {
		return &idriver.IdentityDescriptor{
			Identity:  []byte("hello world"),
			AuditInfo: []byte("audit"),
		}, nil
	}, 10000, nil, NewMetrics(&disabled.Provider{}))

	numRoutines := 4
	wg := sync.WaitGroup{}
	wg.Add(numRoutines)
	for range numRoutines {
		go func() {
			defer wg.Done()

			for range 100 {
				id, err := c.Identity(t.Context(), nil)
				require.NoError(t, err)
				assert.Equal(t, driver.Identity("hello world"), id.Identity)
				assert.Equal(t, []byte("audit"), id.AuditInfo)
			}
		}()
	}
	wg.Wait()
}

// TestFetchIdentityFromBackend verifies backend fetch when audit info doesn't match.
func TestFetchIdentityFromBackend(t *testing.T) {
	expectedIdentity := &idriver.IdentityDescriptor{
		Identity:  []byte("backend identity"),
		AuditInfo: []byte("backend audit"),
	}

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		return expectedIdentity, nil
	}, 10, []byte("cache audit"), NewMetrics(&disabled.Provider{}))

	// Call with different audit info to trigger backend fetch
	identityDescriptor, err := c.Identity(context.Background(), []byte("different audit"))
	require.NoError(t, err)
	assert.Equal(t, expectedIdentity.Identity, identityDescriptor.Identity)
	assert.Equal(t, expectedIdentity.AuditInfo, identityDescriptor.AuditInfo)
}

// TestFetchIdentityFromBackendError verifies error propagation from backend failures.
func TestFetchIdentityFromBackendError(t *testing.T) {
	expectedErr := errors.New("backend error")

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		return nil, expectedErr
	}, 10, []byte("cache audit"), NewMetrics(&disabled.Provider{}))

	// Call with different audit info to trigger backend fetch
	_, err := c.Identity(context.Background(), []byte("different audit"))
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestFetchIdentityFromCacheTimeout verifies on-demand generation after cache timeout.
func TestFetchIdentityFromCacheTimeout(t *testing.T) {
	var callCount atomic.Int32
	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		callCount.Add(1)
		// Simulate slow backend - not strictly needed for the test
		// time.Sleep(10 * time.Millisecond)
		return &idriver.IdentityDescriptor{
			Identity:  []byte("timeout identity"),
			AuditInfo: []byte("timeout audit"),
		}, nil
	}, 0, nil, NewMetrics(&disabled.Provider{})) // cache size 0 to force timeout

	// Set short timeout to trigger timeout path
	c.cacheTimeout = 1 * time.Millisecond

	identityDescriptor, err := c.Identity(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, driver.Identity([]byte("timeout identity")), identityDescriptor.Identity)
	assert.Equal(t, []byte("timeout audit"), identityDescriptor.AuditInfo)
	assert.Equal(t, int32(1), callCount.Load())
}

// TestFetchIdentityFromCacheTimeoutError verifies error handling after cache timeout.
func TestFetchIdentityFromCacheTimeoutError(t *testing.T) {
	expectedErr := errors.New("timeout backend error")

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		return nil, expectedErr
	}, 0, nil, NewMetrics(&disabled.Provider{}))

	// Set short timeout to trigger timeout path
	c.cacheTimeout = 1 * time.Millisecond

	_, err := c.Identity(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestProvisionIdentitiesError verifies provisioning retries after errors.
func TestProvisionIdentitiesError(t *testing.T) {
	var callCount atomic.Int32
	maxCalls := int32(3)

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		// Fail 3 times then succeed
		current := callCount.Add(1)
		if current <= maxCalls {
			return nil, errors.New("provision error")
		}

		return &idriver.IdentityDescriptor{
			Identity:  []byte("success identity"),
			AuditInfo: []byte("success audit"),
		}, nil
	}, 10, nil, NewMetrics(&disabled.Provider{}))

	// Trigger provisioning
	_, err := c.Identity(context.Background(), nil)
	require.NoError(t, err)

	// Wait a bit for provisioning to attempt multiple times
	time.Sleep(50 * time.Millisecond)

	// Verify that provisioning continued after errors
	assert.Greater(t, callCount.Load(), maxCalls)
}

// TestFetchIdentityFromCacheNilEntry verifies backend fallback for nil cache entries.
func TestFetchIdentityFromCacheNilEntry(t *testing.T) {
	var backendCalledCount atomic.Int32

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		backendCalledCount.Add(1)

		return &idriver.IdentityDescriptor{
			Identity:  []byte("backend fallback"),
			AuditInfo: []byte("backend audit"),
		}, nil
	}, 10, nil, NewMetrics(&disabled.Provider{}))

	// Pre-populate the cache with nil before calling Identity()
	// Since cache is buffered, this completes immediately
	c.cache <- nil

	// Small delay to ensure the nil is in the buffer before Identity() reads
	time.Sleep(10 * time.Millisecond)

	identityDescriptor, err := c.Identity(context.Background(), nil)
	require.NoError(t, err)
	assert.Eventually(t, func() bool {
		return backendCalledCount.Load() > 0
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, driver.Identity([]byte("backend fallback")), identityDescriptor.Identity)
}
