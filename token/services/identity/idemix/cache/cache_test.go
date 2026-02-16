/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func TestIdentityCache(t *testing.T) {
	// Use MockLogger to enable debug logging and cover debug code paths
	originalLogger := logger
	defer func() { logger = originalLogger }()
	logger = &logging.MockLogger{}

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

// TestFetchIdentityFromBackend tests the fetchIdentityFromBackend function
func TestFetchIdentityFromBackend(t *testing.T) {
	expectedIdentity := &idriver.IdentityDescriptor{
		Identity:  []byte("backend identity"),
		AuditInfo: []byte("backend audit"),
	}

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		return expectedIdentity, nil
	}, 10, []byte("cache audit"), NewMetrics(&disabled.Provider{}))

	// Call with different audit info to trigger fetchIdentityFromBackend in c.Identity
	identityDescriptor, err := c.Identity(context.Background(), []byte("different audit"))
	assert.NoError(t, err)
	assert.Equal(t, expectedIdentity.Identity, identityDescriptor.Identity)
	assert.Equal(t, expectedIdentity.AuditInfo, identityDescriptor.AuditInfo)
}

// TestFetchIdentityFromBackendError tests error handling in fetchIdentityFromBackend
func TestFetchIdentityFromBackendError(t *testing.T) {
	expectedErr := errors.New("backend error")

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		return nil, expectedErr
	}, 10, []byte("cache audit"), NewMetrics(&disabled.Provider{}))

	// Call with different audit info to trigger fetchIdentityFromBackend in c.Identity,
	// which is supposed to return the error
	_, err := c.Identity(context.Background(), []byte("different audit"))
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestFetchIdentityFromCacheTimeout tests the timeout scenario
func TestFetchIdentityFromCacheTimeout(t *testing.T) {
	// Use MockLogger to enable debug logging and cover debug code paths
	originalLogger := logger
	defer func() { logger = originalLogger }()
	logger = &logging.MockLogger{}

	callCount := 0
	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		callCount++
		// Simulate slow backend - not strictly needed for the test
		// time.Sleep(10 * time.Millisecond)
		return &idriver.IdentityDescriptor{
			Identity:  []byte("timeout identity"),
			AuditInfo: []byte("timeout audit"),
		}, nil
	}, 0, nil, NewMetrics(&disabled.Provider{})) // cache size 0 to force timeout

	// Set a very short timeout to trigger timeout path in c.Identity, calling the above backend
	c.cacheTimeout = 1 * time.Millisecond

	identityDescriptor, err := c.Identity(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, driver.Identity([]byte("timeout identity")), identityDescriptor.Identity)
	assert.Equal(t, []byte("timeout audit"), identityDescriptor.AuditInfo)
	assert.Equal(t, 1, callCount)
}

// TestFetchIdentityFromCacheTimeoutError tests error handling in timeout scenario
func TestFetchIdentityFromCacheTimeoutError(t *testing.T) {
	expectedErr := errors.New("timeout backend error")

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		return nil, expectedErr
	}, 0, nil, NewMetrics(&disabled.Provider{}))

	// Set a very short timeout to trigger timeout path in c.Identity, calling the above backend
	// that is supposed to returning the err
	c.cacheTimeout = 1 * time.Millisecond

	_, err := c.Identity(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestProvisionIdentitiesError tests error handling in provisionIdentities
func TestProvisionIdentitiesError(t *testing.T) {
	callCount := 0
	maxCalls := 3

	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		// provisionIdentities's loop will fail here 3 times until it finally succeeds
		callCount++
		if callCount <= maxCalls {
			return nil, errors.New("provision error")
		}
		return &idriver.IdentityDescriptor{
			Identity:  []byte("success identity"),
			AuditInfo: []byte("success audit"),
		}, nil
	}, 10, nil, NewMetrics(&disabled.Provider{}))

	// Trigger provisioning by provisionIdentities
	_, err := c.Identity(context.Background(), nil)
	assert.NoError(t, err)

	// Wait a bit for provisioning to attempt multiple times
	time.Sleep(50 * time.Millisecond)

	// Verify that provisioning continued after errors
	assert.Greater(t, callCount, maxCalls)
}

// TestFetchIdentityFromCacheNilEntry tests handling of nil entry from cache
func TestFetchIdentityFromCacheNilEntry(t *testing.T) {
	backendCalled := false
	c := NewIdentityCache(func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
		backendCalled = true
		return &idriver.IdentityDescriptor{
			Identity:  []byte("backend fallback"),
			AuditInfo: []byte("backend audit"),
		}, nil
	}, 10, nil, NewMetrics(&disabled.Provider{}))

	// Manually send nil to cache to test nil handling (i.e. calling the above backend)
	c.cache <- nil

	identityDescriptor, err := c.Identity(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, backendCalled)
	assert.Equal(t, driver.Identity([]byte("backend fallback")), identityDescriptor.Identity)
}
