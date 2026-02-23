/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault_test

import (
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	auditmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	tokenmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	ttxmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewVaultProvider_Counterfeiter verifies vault provider initialization.
func TestNewVaultProvider_Counterfeiter(t *testing.T) {
	tokenMgr := &tokenmock.TokenStoreServiceManager{}
	ttxMgr := &ttxmock.TTXStoreServiceManager{}
	auditMgr := &auditmock.AuditStoreServiceManager{}

	provider := vault.NewVaultProvider(tokenMgr, ttxMgr, auditMgr)

	require.NotNil(t, provider)
}

// TestVaultProvider_Vault_Success_Counterfeiter verifies successful vault creation.
func TestVaultProvider_Vault_Success_Counterfeiter(t *testing.T) {
	tokenMgr := &tokenmock.TokenStoreServiceManager{}
	ttxMgr := &ttxmock.TTXStoreServiceManager{}
	auditMgr := &auditmock.AuditStoreServiceManager{}

	network := "test-network"
	channel := "test-channel"
	namespace := "test-namespace"

	mockTokenDB := &tokendb.StoreService{}
	mockTTXDB := &ttxdb.StoreService{}
	mockAuditDB := &auditdb.StoreService{}

	tokenMgr.StoreServiceByTMSIdReturns(mockTokenDB, nil)
	ttxMgr.StoreServiceByTMSIdReturns(mockTTXDB, nil)
	auditMgr.StoreServiceByTMSIdReturns(mockAuditDB, nil)

	provider := vault.NewVaultProvider(tokenMgr, ttxMgr, auditMgr)
	v, err := provider.Vault(network, channel, namespace)

	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 1, tokenMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 1, ttxMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 1, auditMgr.StoreServiceByTMSIdCallCount())
}

// TestVaultProvider_Vault_Cached_Counterfeiter verifies vault caching behavior.
func TestVaultProvider_Vault_Cached_Counterfeiter(t *testing.T) {
	tokenMgr := &tokenmock.TokenStoreServiceManager{}
	ttxMgr := &ttxmock.TTXStoreServiceManager{}
	auditMgr := &auditmock.AuditStoreServiceManager{}

	network := "test-network"
	channel := "test-channel"
	namespace := "test-namespace"

	mockTokenDB := &tokendb.StoreService{}
	mockTTXDB := &ttxdb.StoreService{}
	mockAuditDB := &auditdb.StoreService{}

	tokenMgr.StoreServiceByTMSIdReturns(mockTokenDB, nil)
	ttxMgr.StoreServiceByTMSIdReturns(mockTTXDB, nil)
	auditMgr.StoreServiceByTMSIdReturns(mockAuditDB, nil)

	provider := vault.NewVaultProvider(tokenMgr, ttxMgr, auditMgr)

	// First call - should create vault
	vault1, err := provider.Vault(network, channel, namespace)
	require.NoError(t, err)
	assert.NotNil(t, vault1)

	// Second call - should return cached vault
	vault2, err := provider.Vault(network, channel, namespace)
	require.NoError(t, err)
	assert.NotNil(t, vault2)
	assert.Same(t, vault1, vault2)

	// Should only be called once due to caching
	assert.Equal(t, 1, tokenMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 1, ttxMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 1, auditMgr.StoreServiceByTMSIdCallCount())
}

// TestVaultProvider_Vault_TokenDBError_Counterfeiter verifies error handling for token DB failures.
func TestVaultProvider_Vault_TokenDBError_Counterfeiter(t *testing.T) {
	tokenMgr := &tokenmock.TokenStoreServiceManager{}
	ttxMgr := &ttxmock.TTXStoreServiceManager{}
	auditMgr := &auditmock.AuditStoreServiceManager{}

	network := "test-network"
	channel := "test-channel"
	namespace := "test-namespace"

	expectedErr := errors.New("token db error")
	tokenMgr.StoreServiceByTMSIdReturns(nil, expectedErr)

	provider := vault.NewVaultProvider(tokenMgr, ttxMgr, auditMgr)
	v, err := provider.Vault(network, channel, namespace)

	require.Error(t, err)
	assert.Nil(t, v)
	assert.Contains(t, err.Error(), "failed to get token db")
	assert.Equal(t, 1, tokenMgr.StoreServiceByTMSIdCallCount())
}

// TestVaultProvider_Vault_TTXDBError_Counterfeiter verifies error handling for transaction DB failures.
func TestVaultProvider_Vault_TTXDBError_Counterfeiter(t *testing.T) {
	tokenMgr := &tokenmock.TokenStoreServiceManager{}
	ttxMgr := &ttxmock.TTXStoreServiceManager{}
	auditMgr := &auditmock.AuditStoreServiceManager{}

	network := "test-network"
	channel := "test-channel"
	namespace := "test-namespace"

	mockTokenDB := &tokendb.StoreService{}
	expectedErr := errors.New("ttx db error")

	tokenMgr.StoreServiceByTMSIdReturns(mockTokenDB, nil)
	ttxMgr.StoreServiceByTMSIdReturns(nil, expectedErr)

	provider := vault.NewVaultProvider(tokenMgr, ttxMgr, auditMgr)
	v, err := provider.Vault(network, channel, namespace)

	require.Error(t, err)
	assert.Nil(t, v)
	assert.Contains(t, err.Error(), "failed to get ttx db")
	assert.Equal(t, 1, tokenMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 1, ttxMgr.StoreServiceByTMSIdCallCount())
}

// TestVaultProvider_Vault_AuditDBError_Counterfeiter verifies error handling for audit DB failures.
func TestVaultProvider_Vault_AuditDBError_Counterfeiter(t *testing.T) {
	tokenMgr := &tokenmock.TokenStoreServiceManager{}
	ttxMgr := &ttxmock.TTXStoreServiceManager{}
	auditMgr := &auditmock.AuditStoreServiceManager{}

	network := "test-network"
	channel := "test-channel"
	namespace := "test-namespace"

	mockTokenDB := &tokendb.StoreService{}
	mockTTXDB := &ttxdb.StoreService{}
	expectedErr := errors.New("audit db error")

	tokenMgr.StoreServiceByTMSIdReturns(mockTokenDB, nil)
	ttxMgr.StoreServiceByTMSIdReturns(mockTTXDB, nil)
	auditMgr.StoreServiceByTMSIdReturns(nil, expectedErr)

	provider := vault.NewVaultProvider(tokenMgr, ttxMgr, auditMgr)
	v, err := provider.Vault(network, channel, namespace)

	require.Error(t, err)
	assert.Nil(t, v)
	assert.Contains(t, err.Error(), "failed to get audit db")
	assert.Equal(t, 1, tokenMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 1, ttxMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 1, auditMgr.StoreServiceByTMSIdCallCount())
}

// TestVaultProvider_Vault_MultipleNetworks_Counterfeiter verifies handling of multiple network vaults.
func TestVaultProvider_Vault_MultipleNetworks_Counterfeiter(t *testing.T) {
	tokenMgr := &tokenmock.TokenStoreServiceManager{}
	ttxMgr := &ttxmock.TTXStoreServiceManager{}
	auditMgr := &auditmock.AuditStoreServiceManager{}

	mockTokenDB1 := &tokendb.StoreService{}
	mockTTXDB1 := &ttxdb.StoreService{}
	mockAuditDB1 := &auditdb.StoreService{}

	mockTokenDB2 := &tokendb.StoreService{}
	mockTTXDB2 := &ttxdb.StoreService{}
	mockAuditDB2 := &auditdb.StoreService{}

	tokenMgr.StoreServiceByTMSIdReturnsOnCall(0, mockTokenDB1, nil)
	ttxMgr.StoreServiceByTMSIdReturnsOnCall(0, mockTTXDB1, nil)
	auditMgr.StoreServiceByTMSIdReturnsOnCall(0, mockAuditDB1, nil)

	tokenMgr.StoreServiceByTMSIdReturnsOnCall(1, mockTokenDB2, nil)
	ttxMgr.StoreServiceByTMSIdReturnsOnCall(1, mockTTXDB2, nil)
	auditMgr.StoreServiceByTMSIdReturnsOnCall(1, mockAuditDB2, nil)

	provider := vault.NewVaultProvider(tokenMgr, ttxMgr, auditMgr)

	vault1, err := provider.Vault("network1", "channel1", "namespace1")
	require.NoError(t, err)
	assert.NotNil(t, vault1)

	vault2, err := provider.Vault("network2", "channel2", "namespace2")
	require.NoError(t, err)
	assert.NotNil(t, vault2)

	// Vaults should be different
	assert.NotSame(t, vault1, vault2)
	assert.Equal(t, 2, tokenMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 2, ttxMgr.StoreServiceByTMSIdCallCount())
	assert.Equal(t, 2, auditMgr.StoreServiceByTMSIdCallCount())
}
