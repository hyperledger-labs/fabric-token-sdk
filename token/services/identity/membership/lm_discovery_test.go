/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	fscdriver "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalMembership_Discovery(t *testing.T) {
	ctx := t.Context()

	ip := &mock.IdentityProvider{}
	ip.BindReturns(nil)

	des := &mock.SignerDeserializerManager{}

	iss := &mock.IdentityStoreService{}
	iss.ConfigurationExistsReturns(false, nil)
	iss.AddConfigurationReturns(nil)

	km1 := &mock.KeyManager{}
	km1.EnrollmentIDReturns("e1")
	km1.AnonymousReturns(false)
	km1.IsRemoteReturns(false)
	idDesc1 := &idriver.IdentityDescriptor{Identity: []byte("id1"), AuditInfo: []byte("ai1")}
	km1.IdentityReturns(idDesc1, nil)
	km1.IdentityTypeReturns("typ")

	km2 := &mock.KeyManager{}
	km2.EnrollmentIDReturns("e2")
	km2.AnonymousReturns(false)
	km2.IsRemoteReturns(false)
	idDesc2 := &idriver.IdentityDescriptor{Identity: []byte("id2"), AuditInfo: []byte("ai2")}
	km2.IdentityReturns(idDesc2, nil)
	km2.IdentityTypeReturns("typ")

	kmp := &mock.KeyManagerProvider{}
	kmp.GetReturnsOnCall(0, km1, nil)
	kmp.GetReturnsOnCall(1, km2, nil)

	lm := membership.NewLocalMembership(
		logging.MustGetLogger("test"),
		&mock.Config{},
		[]byte("netid"),
		des,
		iss,
		"testType",
		false,
		ip,
		kmp,
	)

	// Initially empty Load
	iss.IteratorConfigurationsReturns(&mock.IdentityConfigurationIterator{}, nil)
	err := lm.Load(ctx, nil, nil)
	require.NoError(t, err)

	ids, _ := lm.IDs()
	assert.Empty(t, ids)

	// Mock DB update
	newConfig := idriver.IdentityConfiguration{ID: "new", URL: "/tmp/new", Type: "testType"}
	iter := &mock.IdentityConfigurationIterator{}
	iter.NextReturnsOnCall(0, &newConfig, nil)
	iter.NextReturnsOnCall(1, nil, nil)
	iss.IteratorConfigurationsReturns(iter, nil)

	// Discovery via GetIdentityInfo
	info, err := lm.GetIdentityInfo(ctx, "new", nil)
	require.NoError(t, err)
	assert.NotNil(t, info)

	ids, _ = lm.IDs()
	assert.Contains(t, ids, "new")
}

func TestLocalMembership_DefaultOverride(t *testing.T) {
	ctx := t.Context()

	ip := &mock.IdentityProvider{}
	ip.BindReturns(nil)

	des := &mock.SignerDeserializerManager{}

	iss := &mock.IdentityStoreService{}
	iss.ConfigurationExistsReturns(false, nil)
	iss.AddConfigurationReturns(nil)

	km1 := &mock.KeyManager{}
	km1.EnrollmentIDReturns("e1")
	km1.IdentityReturns(&idriver.IdentityDescriptor{Identity: []byte("id1")}, nil)
	km1.IdentityTypeReturns("typ")

	km2 := &mock.KeyManager{}
	km2.EnrollmentIDReturns("e2")
	km2.IdentityReturns(&idriver.IdentityDescriptor{Identity: []byte("id2")}, nil)
	km2.IdentityTypeReturns("typ")

	kmp := &mock.KeyManagerProvider{}
	kmp.GetReturnsOnCall(0, km1, nil)
	kmp.GetReturnsOnCall(1, km2, nil)

	lm := membership.NewLocalMembership(
		logging.MustGetLogger("test"),
		&mock.Config{},
		[]byte("netid"),
		des,
		iss,
		"testType",
		false,
		ip,
		kmp,
	)

	// Load with one identity as default
	identities := []idriver.ConfiguredIdentity{{ID: "id1", Path: "/tmp/id1", Default: true}}
	iss.IteratorConfigurationsReturns(&mock.IdentityConfigurationIterator{}, nil)
	err := lm.Load(ctx, identities, nil)
	require.NoError(t, err)
	assert.Equal(t, "id1", lm.GetDefaultIdentifier())

	// Discovery of a new identity (it was previously default: true, but now we don't support it)
	newConfig := idriver.IdentityConfiguration{ID: "id2", URL: "/tmp/id2", Type: "testType"}
	iter := &mock.IdentityConfigurationIterator{}
	iter.NextReturnsOnCall(0, &newConfig, nil)
	iter.NextReturnsOnCall(1, nil, nil)
	iss.IteratorConfigurationsReturns(iter, nil)

	_, err = lm.GetIdentityInfo(ctx, "id2", nil)
	require.NoError(t, err)
	// Default should NOT change
	assert.Equal(t, "id1", lm.GetDefaultIdentifier())
}

func TestLocalMembership_DoubleCheckedLocking(t *testing.T) {
	ctx := t.Context()

	ip := &mock.IdentityProvider{}
	des := &mock.SignerDeserializerManager{}
	iss := &mock.IdentityStoreService{}
	iss.ConfigurationExistsReturns(false, nil)
	iss.AddConfigurationReturns(nil)

	km := &mock.KeyManager{}
	km.EnrollmentIDReturns("e1")
	km.IdentityReturns(&idriver.IdentityDescriptor{Identity: []byte("id1")}, nil)
	km.IdentityTypeReturns("typ")

	kmp := &mock.KeyManagerProvider{}
	kmp.GetReturns(km, nil)

	lm := membership.NewLocalMembership(
		logging.MustGetLogger("test"),
		&mock.Config{},
		[]byte("netid"),
		des,
		iss,
		"testType",
		false,
		ip,
		kmp,
	)

	iss.IteratorConfigurationsReturns(&mock.IdentityConfigurationIterator{}, nil)
	_ = lm.Load(ctx, nil, nil)

	newConfig := idriver.IdentityConfiguration{ID: "new", URL: "/tmp/new", Type: "testType"}

	// We want to ensure that refresh happens only once even if multiple goroutines call it
	var refreshCount int
	iss.IteratorConfigurationsStub = func(ctx context.Context, typ string) (membership.IdentityConfigurationIterator, error) {
		refreshCount++
		iter := &mock.IdentityConfigurationIterator{}
		iter.NextReturnsOnCall(0, &newConfig, nil)
		iter.NextReturnsOnCall(1, nil, nil)

		return iter, nil
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = lm.GetIdentityInfo(ctx, "new", nil)
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, refreshCount)
}

func TestLocalMembership_Notifier(t *testing.T) {
	ctx := t.Context()

	ip := &mock.IdentityProvider{}
	des := &mock.SignerDeserializerManager{}
	iss := &mock.IdentityStoreService{}
	iss.ConfigurationExistsReturns(false, nil)
	iss.AddConfigurationReturns(nil)

	notifier := &mock.IdentityNotifier{}
	iss.NotifierReturns(notifier, nil)

	km := &mock.KeyManager{}
	km.EnrollmentIDReturns("e1")
	km.IdentityReturns(&idriver.IdentityDescriptor{Identity: []byte("id1")}, nil)
	km.IdentityTypeReturns("typ")

	kmp := &mock.KeyManagerProvider{}
	kmp.GetReturns(km, nil)

	lm := membership.NewLocalMembership(
		logging.MustGetLogger("test"),
		&mock.Config{},
		[]byte("netid"),
		des,
		iss,
		"testType",
		false,
		ip,
		kmp,
	)

	iss.IteratorConfigurationsReturns(&mock.IdentityConfigurationIterator{}, nil)

	var mu sync.Mutex
	var subCallback fscdriver.TriggerCallback
	notifier.SubscribeStub = func(callback fscdriver.TriggerCallback) error {
		mu.Lock()
		defer mu.Unlock()
		subCallback = callback

		return nil
	}

	err := lm.Load(ctx, nil, nil)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return subCallback != nil
	}, 2*time.Second, 100*time.Millisecond)

	newConfig := idriver.IdentityConfiguration{ID: "new", URL: "/tmp/new", Type: "testType"}
	iss.GetConfigurationReturns(&newConfig, nil)

	// Simulate notification
	mu.Lock()
	callback := subCallback
	mu.Unlock()
	callback(idriver.Insert, map[idriver.ColumnKey]string{
		"id":   "new",
		"type": "testType",
		"url":  "/tmp/new",
	})

	// Wait for background processing (though handleConfig is called synchronously in the callback in my implementation,
	// but the callback itself is called from the notifier which might be background).
	// In my lm.go, the callback is executed by the notifier loop.

	assert.Eventually(t, func() bool {
		ids, _ := lm.IDs()

		return slices.Contains(ids, "new")
	}, 2*time.Second, 100*time.Millisecond)
}

func TestLocalMembership_Close(t *testing.T) {
	ctx := t.Context()

	ip := &mock.IdentityProvider{}
	des := &mock.SignerDeserializerManager{}
	iss := &mock.IdentityStoreService{}
	iss.ConfigurationExistsReturns(false, nil)
	iss.AddConfigurationReturns(nil)
	iss.IteratorConfigurationsReturns(&mock.IdentityConfigurationIterator{}, nil)

	kmp := &mock.KeyManagerProvider{}

	lm := membership.NewLocalMembership(
		logging.MustGetLogger("test"),
		&mock.Config{},
		[]byte("netid"),
		des,
		iss,
		"testType",
		false,
		ip,
		kmp,
	)

	err := lm.Load(ctx, nil, nil)
	require.NoError(t, err)

	// Close the membership
	lm.Close()

	// Should be safe to call multiple times
	lm.Close()
}
