/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/require"
)

func TestServiceManager(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTMSProv := &mockTMSProvider{}
	mockStoreServProv := &mockStoreServiceManager{
		StoreServiceByTMSIdReturns: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
	}
	mockNetProv := &mockNetworkProvider{}
	mockPub := &mockPublisher{}

	manager := NewServiceManager(mockTMSProv, mockStoreServProv, mockNetProv, mockPub)
	require.NotNil(t, manager)

	// Test ServiceByTMSId
	service, err := manager.ServiceByTMSId(tmsID)
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, mockTMSProv, service.TMSProvider)
	assert.Equal(t, mockNetProv, service.NetworkProvider)

	// Test GetService helper
	sp := &mockServiceProvider{
		GetServiceReturns: manager,
	}
	service2, err := GetService(sp, tmsID)
	require.NoError(t, err)
	assert.Equal(t, service, service2)

	// Test GetService error
	spErr := &mockServiceProvider{
		GetServiceError: assert.AnError,
	}
	_, err = GetService(spErr, tmsID)
	assert.Error(t, err)
}
